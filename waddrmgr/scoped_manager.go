package waddrmgr

import (
	"fmt"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/czh0526/btc-wallet/internal/zero"
	"github.com/czh0526/btc-wallet/walletdb"
	"github.com/lightninglabs/neutrino/cache/lru"
	"sync"
)

const (
	defaultPrivKeyCacheSize = 10_000
)

type KeyScope struct {
	Purpose uint32
	Coin    uint32
}

type ScopeAddrSchema struct {
	ExternalAddrType AddressType
	InternalAddrType AddressType
}

var (
	KeyScopeBIP0049Plus = KeyScope{
		Purpose: 49,
		Coin:    0,
	}

	KeyScopeBIP0084 = KeyScope{
		Purpose: 84,
		Coin:    0,
	}

	KeyScopeBIP0086 = KeyScope{
		Purpose: 86,
		Coin:    0,
	}

	KeyScopeBIP0044 = KeyScope{
		Purpose: 44,
		Coin:    0,
	}

	DefaultKeyScopes = []KeyScope{
		KeyScopeBIP0049Plus,
		KeyScopeBIP0084,
		KeyScopeBIP0086,
		KeyScopeBIP0044,
	}

	ScopeAddrMap = map[KeyScope]ScopeAddrSchema{
		KeyScopeBIP0049Plus: {
			ExternalAddrType: NestedWitnessPubKey,
			InternalAddrType: WitnessPubKey,
		},
		KeyScopeBIP0084: {
			ExternalAddrType: WitnessPubKey,
			InternalAddrType: WitnessPubKey,
		},
		KeyScopeBIP0086: {
			ExternalAddrType: TaprootPubKey,
			InternalAddrType: TaprootPubKey,
		},
		KeyScopeBIP0044: {
			ExternalAddrType: PubKeyHash,
			InternalAddrType: PubKeyHash,
		},
	}
)

type unlockDeriveInfo struct {
	managedAddr ManagedAddress
	branch      uint32
	index       uint32
}

type DerivationPath struct {
	InternalAccount      uint32
	Account              uint32
	Branch               uint32
	Index                uint32
	MasterKeyFingerprint uint32
}

type cachedKey struct {
	key btcec.PrivateKey
}

func (c *cachedKey) Size() (uint64, error) {
	return 1, nil
}

type ScopedKeyManager struct {
	scope          KeyScope
	addrSchema     ScopeAddrSchema
	rootManager    *Manager
	addrs          map[addrKey]ManagedAddress
	acctInfo       map[uint32]*accountInfo
	deriveOnUnlock []*unlockDeriveInfo
	privKeyCache   *lru.Cache[DerivationPath, *cachedKey]

	mtx sync.RWMutex
}

func (s *ScopedKeyManager) AddrSchema() ScopeAddrSchema {
	return s.addrSchema
}

func (s *ScopedKeyManager) Scope() KeyScope {
	return s.scope
}

func (s *ScopedKeyManager) Close() {
	fmt.Println("ScopedKeyManager::Close() was not implemented yet.")
}

func (s *ScopedKeyManager) deriveKeyFromPath(ns walletdb.ReadBucket,
	internalAccount, branch, index uint32, private bool) (
	*hdkeychain.ExtendedKey, *hdkeychain.ExtendedKey, uint32, error) {

	acctInfo, err := s.loadAccountInfo(ns, internalAccount)
	if err != nil {
		return nil, nil, 0, err
	}

	addrKey, err := s.deriveKey(acctInfo, branch, index, private)
	if err != nil {
		return nil, nil, 0, err
	}

	acctKey := acctInfo.acctKeyPub
	if private {
		acctKey = acctInfo.acctKeyPriv
	}

	return addrKey, acctKey, acctInfo.masterKeyFingerprint, nil
}

// 使用 acctInfo 内的密钥， 根据 branch + index 构造一个 ExtendedKey
// ManagedAddress_Key
func (s *ScopedKeyManager) deriveKey(acctInfo *accountInfo,
	branch, index uint32, private bool) (*hdkeychain.ExtendedKey, error) {

	acctKey := acctInfo.acctKeyPub
	if private {
		acctKey = acctInfo.acctKeyPriv
	}

	branchKey, err := acctKey.DeriveNonStandard(branch)
	if err != nil {
		str := fmt.Sprintf("failed to derive extended key branch %d", branch)
		return nil, managerError(ErrKeyChain, str, err)
	}

	addressKey, err := branchKey.DeriveNonStandard(index)

	branchKey.Zero()
	if err != nil {
		str := fmt.Sprintf("failed to derive child extended key -- "+
			"branch `%d`, child `%d`", branch, index)
		return nil, managerError(ErrKeyChain, str, err)
	}

	return addressKey, nil
}

func (s *ScopedKeyManager) keyToManaged(derivedKey *hdkeychain.ExtendedKey,
	derivationPath DerivationPath, acctInfo *accountInfo) (ManagedAddress, error) {

	internal := derivationPath.Branch == InternalBranch
	addrType := s.accountAddrType(acctInfo, internal)

	ma, err := newManagedAddressFromExtKey(
		s, derivationPath, derivedKey, addrType, acctInfo)
	defer derivedKey.Zero()
	if err != nil {
		return nil, err
	}

	if !derivedKey.IsPrivate() {
		info := unlockDeriveInfo{
			managedAddr: ma,
			branch:      derivationPath.Branch,
			index:       derivationPath.Index,
		}
		s.deriveOnUnlock = append(s.deriveOnUnlock, &info)
	}

	if derivationPath.Branch == InternalBranch {
		ma.internal = true
	}

	return ma, nil
}

func (s *ScopedKeyManager) accountAddrType(acctInfo *accountInfo, internal bool) AddressType {
	addrSchema := s.addrSchema
	if acctInfo.addrSchema != nil {
		addrSchema = *acctInfo.addrSchema
	}

	if internal {
		return addrSchema.InternalAddrType
	}
	return addrSchema.ExternalAddrType
}

// 使用 rootManager 的密钥恢复 account 的密钥，
// 并存入 accountInfo 中
func (s *ScopedKeyManager) loadAccountInfo(ns walletdb.ReadBucket,
	account uint32) (*accountInfo, error) {

	// 检查缓存
	if acctInfo, ok := s.acctInfo[account]; ok {
		return acctInfo, nil
	}

	// 取出 db 中的 accountInfo
	rowInterface, err := fetchAccountInfo(ns, &s.scope, account)
	if err != nil {
		return nil, maybeConvertDbError(err)
	}

	// 使用 cryptoKey 恢复 encryptedKey 的中的 ExtendedKey
	decryptKey := func(cryptoKey EncryptorDecryptor, encryptedKey []byte) (*hdkeychain.ExtendedKey, error) {
		serializedKey, err := cryptoKey.Decrypt(encryptedKey)
		if err != nil {
			return nil, err
		}

		return hdkeychain.NewKeyFromString(string(serializedKey))
	}

	watchOnly := s.rootManager.watchOnly()
	hasPrivateKey := !s.rootManager.isLocked() && !watchOnly

	var acctInfo *accountInfo
	switch row := rowInterface.(type) {
	case *dbDefaultAccountRow:
		acctInfo = &accountInfo{
			acctName:          row.name,
			acctType:          row.acctType,
			acctKeyEncrypted:  row.privKeyEncrypted,
			nextExternalIndex: row.nextExternalIndex,
			nextInternalIndex: row.nextInternalIndex,
		}

		// 恢复 row 中的公钥
		acctInfo.acctKeyPub, err = decryptKey(
			s.rootManager.cryptoKeyPub, row.pubKeyEncrypted)
		if err != nil {
			str := fmt.Sprintf("failed to decrypted to decrypt public key for account %d", account)
			return nil, managerError(ErrCrypto, str, err)
		}

		// 恢复 row 中的私钥
		if hasPrivateKey {
			acctInfo.acctKeyPriv, err = decryptKey(
				s.rootManager.cryptoKeyPriv, row.privKeyEncrypted)
		}

	case *dbWatchOnlyAccountRow:
		acctInfo = &accountInfo{
			acctName:             row.name,
			acctType:             row.acctType,
			nextExternalIndex:    row.nextExternalIndex,
			nextInternalIndex:    row.nextInternalIndex,
			addrSchema:           row.addrSchema,
			masterKeyFingerprint: row.masterKeyFingerprint,
		}

		// 恢复 row 中的公钥
		acctInfo.acctKeyPub, err = decryptKey(
			s.rootManager.cryptoKeyPub, row.pubKeyEncrypted)
		if err != nil {
			str := fmt.Sprintf("failed to decrypted to decrypt public key for account %d", account)
			return nil, managerError(ErrCrypto, str, err)
		}

		hasPrivateKey = false

	default:
		str := fmt.Sprintf("unsupported account type %T", row)
		return nil, managerError(ErrDatabase, str, nil)
	}

	branch, index := ExternalBranch, acctInfo.nextExternalIndex
	if index > 0 {
		index--
	}
	lastIntAddrPath := DerivationPath{
		InternalAccount:      account,
		Account:              acctInfo.acctKeyPub.ChildIndex(),
		Branch:               branch,
		Index:                index,
		MasterKeyFingerprint: acctInfo.masterKeyFingerprint,
	}
	lastIntKey, err := s.deriveKey(acctInfo, branch, index, hasPrivateKey)
	if err != nil {
		return nil, err
	}
	lastIntAddr, err := s.keyToManaged(lastIntKey, lastIntAddrPath, acctInfo)
	if err != nil {
		return nil, err
	}
	acctInfo.lastInternalAddr = lastIntAddr

	s.acctInfo[account] = acctInfo
	return acctInfo, nil
}

func (s *ScopedKeyManager) nextAddresses(ns walletdb.ReadWriteBucket,
	account uint32, numAddresses uint32, internal bool) (
	[]ManagedAddress, error) {

	acctInfo, err := s.loadAccountInfo(ns, account)
	if err != nil {
		return nil, err
	}

	acctKey := acctInfo.acctKeyPub
	watchOnly := s.rootManager.WatchOnly() || len(acctInfo.acctKeyEncrypted) == 0
	if !s.rootManager.IsLocked() && !watchOnly {
		acctKey = acctInfo.acctKeyPriv
	}

	branchNum, nextIndex := ExternalBranch, acctInfo.nextExternalIndex
	if internal {
		branchNum = InternalBranch
		nextIndex = acctInfo.nextInternalIndex
	}

	addrType := s.accountAddrType(acctInfo, internal)

	if numAddresses > MaxAddressesPerAccount || nextIndex+numAddresses > MaxAddressesPerAccount {
		str := fmt.Sprintf("%d new addresses would exceed the maximum number of addresses per account %d",
			numAddresses, MaxAddressesPerAccount)
		return nil, managerError(ErrTooManyAddresses, str, nil)
	}

	branchKey, err := acctKey.DeriveNonStandard(branchNum)
	if err != nil {
		str := fmt.Sprintf("failed to derive extended key branch %d", branchNum)
		return nil, managerError(ErrKeyChain, str, err)
	}
	defer branchKey.Zero()

}

func (s *ScopedKeyManager) fetchUsed(ns walletdb.ReadBucket,
	addressID []byte) bool {

	return fetchAddressUsed(ns, &s.scope, addressID)
}

func (s *ScopedKeyManager) MarkUsed(ns walletdb.ReadWriteBucket,
	address btcutil.Address) error {

	addressID := address.ScriptAddress()
	err := markAddressUsed(ns, &s.scope, addressID)
	if err != nil {
		return maybeConvertDbError(err)
	}

	s.mtx.Lock()
	delete(s.addrs, addrKey(addressID))
	s.mtx.Unlock()

	return nil
}

func (s *ScopedKeyManager) NewAccount(ns walletdb.ReadWriteBucket, name string) (uint32, error) {
	if s.rootManager.WatchOnly() {
		return 0, managerError(ErrWatchingOnly, errWatchingOnly, nil)
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.rootManager.IsLocked() {
		return 0, managerError(ErrLocked, errLocked, nil)
	}

	account, err := fetchLastAccount(ns, &s.scope)
	if err != nil {
		return 0, err
	}
	account++

	if err := s.newAccount(ns, account, name); err != nil {
		return 0, err
	}

	return account, nil
}

func (s *ScopedKeyManager) newAccount(ns walletdb.ReadWriteBucket,
	account uint32, name string) error {

	if err := ValidateAccountName(name); err != nil {
		return err
	}

	_, err := s.lookupAccount(ns, name)
	if err != nil {
		str := "account with the same name already exists"
		return managerError(ErrAccountNotFound, str, err)
	}

	_, coinTypePrivEnc, err := fetchCoinTypeKeys(ns, &s.scope)
	if err != nil {
		return err
	}

	serializedKeyPriv, err := s.rootManager.cryptoKeyPriv.Decrypt(coinTypePrivEnc)
	if err != nil {
		str := "failed to decrypt cointype serialized private key"
		return managerError(ErrLocked, str, err)
	}
	coinTypeKeyPriv, err := hdkeychain.NewKeyFromString(string(serializedKeyPriv))
	zero.Bytes(serializedKeyPriv)
	if err != nil {
		str := "failed to create cointype extended private key"
		return managerError(ErrKeyChain, str, err)
	}

	acctKeyPriv, err := deriveAccountKey(coinTypeKeyPriv, account)
	coinTypeKeyPriv.Zero()
	if err != nil {
		str := "failed to convert private key for account"
		return managerError(ErrKeyChain, str, err)
	}

	acctKeyPub, err := acctKeyPriv.Neuter()
	if err != nil {
		str := "failed to convert public key for account"
		return managerError(ErrKeyChain, str, err)
	}

	acctPubEnc, err := s.rootManager.cryptoKeyPub.Encrypt([]byte(acctKeyPub.String()))
	if err != nil {
		str := "failed to encrypt public key for account"
		return managerError(ErrCrypto, str, err)
	}
	acctPrivEnc, err := s.rootManager.cryptoKeyPriv.Encrypt([]byte(acctKeyPriv.String()))
	if err != nil {
		str := "failed to encrypt private key for account"
		return managerError(ErrCrypto, str, err)
	}

	err = putDefaultAccountInfo(ns, &s.scope,
		account, acctPubEnc, acctPrivEnc, 0, 0, name)
	if err != nil {
		return err
	}

	return putLastAccount(ns, &s.scope, account)
}

func (s *ScopedKeyManager) LookupAccount(ns walletdb.ReadBucket, name string) (uint32, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	return s.lookupAccount(ns, name)
}

func (s *ScopedKeyManager) lookupAccount(ns walletdb.ReadBucket, name string) (uint32, error) {
	return fetchAccountByName(ns, &s.scope, name)
}
