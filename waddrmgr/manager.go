package waddrmgr

import (
	"crypto/rand"
	"crypto/sha512"
	"fmt"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/czh0526/btc-wallet/internal/zero"
	"github.com/czh0526/btc-wallet/snacl"
	"github.com/czh0526/btc-wallet/walletdb"
	"github.com/lightninglabs/neutrino/cache/lru"
	"sync"
	"time"
)

const (
	DefaultAccountNum  = 0
	defaultAccountName = "default"

	maxCoinType = hdkeychain.HardenedKeyStart - 1

	MaxAccountNum = hdkeychain.HardenedKeyStart - 2

	MaxAddressesPerAccount = hdkeychain.HardenedKeyStart - 1

	saltSize = 32

	ExternalBranch uint32 = 0
	InternalBranch uint32 = 1

	ImportedAddrAccount     = MaxAccountNum + 1
	ImportedAddrAccountName = "imported"
)

type addrKey string

var (
	DefaultScryptOptions = ScryptOptions{
		N: 262144,
		R: 8,
		P: 1,
	}
)

var (
	secretKeyGen    = defaultNewSecretKey
	secretKeyGenMtx sync.RWMutex

	newCryptoKey = defaultNewCryptoKey
)

func defaultNewSecretKey(passphrase *[]byte,
	config *ScryptOptions) (*snacl.SecretKey, error) {
	return snacl.NewSecretKey(passphrase, config.N, config.R, config.P)
}

type Manager struct {
	mtx sync.RWMutex

	chainParams         *chaincfg.Params
	scopedManagers      map[KeyScope]*ScopedKeyManager
	externalAddrSchemas map[AddressType][]KeyScope
	internalAddrSchemas map[AddressType][]KeyScope
	watchingOnly        bool

	masterKeyPub  *snacl.SecretKey
	masterKeyPriv *snacl.SecretKey

	cryptoKeyPub EncryptorDecryptor

	cryptoKeyPriv          EncryptorDecryptor
	cryptoKeyPrivEncrypted []byte

	cryptoKeyScript          EncryptorDecryptor
	cryptoKeyScriptEncrypted []byte

	privPassphraseSalt   [saltSize]byte
	hashedPrivPassphrase [sha512.Size]byte

	locked bool
	closed bool
}

type ScryptOptions struct {
	N, R, P int
}

var FastScryptOptions = ScryptOptions{
	N: 16,
	R: 8,
	P: 1,
}

// Create 创建 Manager
/*

    masterKeyPub    |   "mpub" -> Marshal
 ___________________|___________________________
    masterKeyPriv   |   "mpriv" -> Marshal
 ___________________|___________________________



                    |    masterKeyPub   |    masterKeyPriv
  __________________|___________________|________________________
     cryptoKeyPub   |   "cpub" -> Enc   |
  __________________|___________________|________________________
     cryptoKeyPriv  |                   |  "cpriv" -> Enc
  __________________|___________________|________________________
    cryptoKeyScript |                   |  "cscript" -> Enc
  __________________|___________________|________________________



                    |    cryptoKeyPub    |     cryptoKeyPriv
  __________________|____________________|________________________
         rootKey    |   "mhdpub" -> Enc  |    "mhdpriv" -> Enc
  __________________|____________________|________________________

*/
func Create(ns walletdb.ReadWriteBucket, rootKey *hdkeychain.ExtendedKey,
	pubPassphrase, privPassphrase []byte,
	chainParams *chaincfg.Params, config *ScryptOptions,
	birthday time.Time) error {

	isWatchingOnly := rootKey == nil

	exists := managerExists(ns)
	if exists {
		return managerError(ErrAlreadyExists, errAlreadyExists, nil)
	}

	if !isWatchingOnly && len(privPassphrase) == 0 {
		str := "private passphrase may not be empty"
		return managerError(ErrEmptyPassphrase, str, nil)
	}

	defaultScope := map[KeyScope]ScopeAddrSchema{}
	if !isWatchingOnly {
		defaultScope = ScopeAddrMap
	}
	fmt.Println("create manager ns =>")
	if err := createManagerNS(ns, defaultScope); err != nil {
		return maybeConvertDbError(err)
	}
	fmt.Println()

	if config == nil {
		config = &DefaultScryptOptions
	}

	// Secret-Key for pub
	fmt.Println("new secret key from pubPassphrase")
	masterKeyPub, err := newSecretKey(&pubPassphrase, config)
	if err != nil {
		str := "failed to master public key"
		return managerError(ErrCrypto, str, err)
	}
	fmt.Println()

	// Crypto-Key for pub
	fmt.Println("new crypto key")
	cryptoKeyPub, err := newCryptoKey()
	if err != nil {
		str := "failed to generate crypto public key"
		return managerError(ErrCrypto, str, err)
	}
	fmt.Println()

	// encoded Crypto-Key for pub
	fmt.Println("encrypt `crypto key` => ")
	cryptoKeyPubEnc, err := masterKeyPub.Encrypt(cryptoKeyPub.Bytes())
	if err != nil {
		str := "failed to encrypt crypto public key"
		return managerError(ErrCrypto, str, err)
	}
	fmt.Printf("cryptoKeyPubEnc => %v \n", cryptoKeyPubEnc)
	fmt.Println()

	//createdAt := &BlockStamp{
	//	Hash:      *chainParams.GenesisHash,
	//	Height:    0,
	//	Timestamp: chainParams.GenesisBlock.Header.Timestamp,
	//}

	//syncInfo := newSyncState(createdAt, createdAt)

	pubParams := masterKeyPub.Marshal()

	var privParams []byte
	var masterKeyPriv *snacl.SecretKey
	var cryptoKeyPrivEnc []byte
	var cryptoKeyScriptEnc []byte
	if !isWatchingOnly {
		// Secret-Key for priv
		fmt.Println("new secret key from privPassphrase")
		masterKeyPriv, err = newSecretKey(&privPassphrase, config)
		if err != nil {
			str := "failed to master private key"
			return managerError(ErrCrypto, str, err)
		}
		defer masterKeyPriv.Zero()
		fmt.Println()

		var privPassphraseSalt [saltSize]byte
		_, err = rand.Read(privPassphraseSalt[:])
		if err != nil {
			str := "failed to read random source for passphrase salt"
			return managerError(ErrCrypto, str, err)
		}

		// Crypto-Key for priv
		cryptoKeyPriv, err := newCryptoKey()
		if err != nil {
			str := "failed to generate crypto private key"
			return managerError(ErrCrypto, str, err)
		}
		defer cryptoKeyPriv.Zero()

		// Crypto-Key for script
		cryptoKeyScript, err := newCryptoKey()
		if err != nil {
			str := "failed to generate crypto script key"
			return managerError(ErrCrypto, str, err)
		}
		defer cryptoKeyScript.Zero()

		// 用 private Secret Key 加密 private Crypto Key
		cryptoKeyPrivEnc, err = masterKeyPriv.Encrypt(cryptoKeyPriv.Bytes())
		if err != nil {
			str := "failed to encrypt crypto private key"
			return managerError(ErrCrypto, str, err)
		}

		// 用 private Secret Key 加密 script Crypto Key
		cryptoKeyScriptEnc, err = masterKeyPriv.Encrypt(cryptoKeyScript.Bytes())
		if err != nil {
			str := "failed to encrypt crypto script key"
			return managerError(ErrCrypto, str, err)
		}

		rootPubKey, err := rootKey.Neuter()
		if err != nil {
			str := "failed to neuter master extended key"
			return managerError(ErrKeyChain, str, err)
		}

		for _, defaultScope := range DefaultKeyScopes {
			fmt.Printf("create manager key scope => `%v` \n", defaultScope)
			err := createManagerKeyScope(
				ns, defaultScope, rootKey, cryptoKeyPub, cryptoKeyPriv)
			if err != nil {
				return maybeConvertDbError(err)
			}
			fmt.Println()
		}

		// 保存 root key
		masterHDPrivKeyEnc, err := cryptoKeyPriv.Encrypt([]byte(rootKey.String()))
		if err != nil {
			return maybeConvertDbError(err)
		}
		masterHDPubKeyEnc, err := cryptoKeyPub.Encrypt([]byte(rootPubKey.String()))
		if err != nil {
			return maybeConvertDbError(err)
		}
		err = putMasterHDKeys(ns, masterHDPrivKeyEnc, masterHDPubKeyEnc)
		if err != nil {
			return maybeConvertDbError(err)
		}

		privParams = masterKeyPriv.Marshal()
	}

	// 保存 master key
	err = putMasterKeyParams(ns, pubParams, privParams)
	if err != nil {
		return maybeConvertDbError(err)
	}

	// 保存 crypto key
	err = putCryptoKeys(ns, cryptoKeyPubEnc, cryptoKeyPrivEnc, cryptoKeyScriptEnc)
	if err != nil {
		return maybeConvertDbError(err)
	}

	err = putWatchingOnly(ns, isWatchingOnly)
	if err != nil {
		return maybeConvertDbError(err)
	}

	//err = PutSyncedTo(ns, &syncInfo.syncedTo)
	//if err != nil {
	//	return maybeConvertDbError(err)
	//}
	//
	//err = putStartBlock(ns, &syncInfo.startBlock)
	//if err != nil {
	//	return maybeConvertDbError(err)
	//}

	return putBirthday(ns, birthday.Add(-48*time.Hour))
}

func Open(ns walletdb.ReadBucket, pubPassphrase []byte,
	chainParams *chaincfg.Params) (*Manager, error) {

	exists := managerExists(ns)
	if !exists {
		str := "the specified address manager does not exist"
		return nil, managerError(ErrNoExist, str, nil)
	}

	return loadManager(ns, pubPassphrase, chainParams)
}

func (m *Manager) Close() {
	m.closed = true
}

func (m *Manager) WatchOnly() bool {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	return m.watchOnly()
}

func (m *Manager) watchOnly() bool {
	return m.watchingOnly
}

func (m *Manager) IsLocked() bool {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	return m.isLocked()
}

func (m *Manager) isLocked() bool {
	return m.locked
}

func (m *Manager) lock() {
	for _, manager := range m.scopedManagers {
		for _, acctInfo := range manager.acctInfo {
			if acctInfo.acctKeyPriv != nil {
				acctInfo.acctKeyPriv.Zero()
			}
			acctInfo.acctKeyPriv = nil
		}
	}

	for _, manager := range m.scopedManagers {
		for _, ma := range manager.addrs {
			switch addr := ma.(type) {
			case *managedAddress:
				addr.lock()
			case *scriptAddress:
				addr.lock()
			}
		}
	}

	m.cryptoKeyScript.Zero()
	m.cryptoKeyPriv.Zero()
	m.masterKeyPriv.Zero()
	zero.Bytea64(&m.hashedPrivPassphrase)

	m.locked = true
}

func (m *Manager) Unlock(ns walletdb.ReadBucket, passphrase []byte) error {
	if m.watchingOnly {
		return managerError(ErrWatchingOnly, errWatchingOnly, nil)
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()

	if !m.locked {
		saltedPassphrase := append(m.privPassphraseSalt[:],
			passphrase...)
		hashedPassphrase := sha512.Sum512(saltedPassphrase)
		zero.Bytes(saltedPassphrase)
		if hashedPassphrase != m.hashedPrivPassphrase {
			m.lock()
			str := "invalid passphrase for master private key"
			return managerError(ErrWrongPassphrase, str, nil)
		}
		return nil
	}

	// 解锁 masterKeyPriv
	if err := m.masterKeyPriv.DeriveKey(&passphrase); err != nil {
		m.lock()
		if err == snacl.ErrInvalidPassword {
			str := "invalid passphrase for master private key"
			return managerError(ErrWrongPassphrase, str, nil)
		}

		str := "failed to derive master private key"
		return managerError(ErrCrypto, str, err)
	}

	decryptedKey, err := m.masterKeyPriv.Decrypt(m.cryptoKeyPrivEncrypted)
	if err != nil {
		m.lock()
		str := "failed to decrypt crypto private key"
		return managerError(ErrCrypto, str, err)
	}
	m.cryptoKeyPriv.CopyBytes(decryptedKey)
	zero.Bytes(decryptedKey)

	for _, manager := range m.scopedManagers {
		for account, acctInfo := range manager.acctInfo {
			decrypted, err := m.cryptoKeyPriv.Decrypt(acctInfo.acctKeyEncrypted)
			if err != nil {
				m.lock()
				str := fmt.Sprintf("failed to decrypt account `%d` private key", account)
				return managerError(ErrCrypto, str, err)
			}

			acctKeyPriv, err := hdkeychain.NewKeyFromString(string(decrypted))
			zero.Bytes(decrypted)
			if err != nil {
				m.lock()
				str := fmt.Sprintf("failed to regenerate account `%d` extended key", account)
				return managerError(ErrKeyChain, str, err)
			}
			acctInfo.acctKeyPriv = acctKeyPriv
		}

		for _, info := range manager.deriveOnUnlock {
			addressKey, _, _, err := manager.deriveKeyFromPath(
				ns, info.managedAddr.InternalAccount(),
				info.branch, info.index, true)
			if err != nil {
				m.lock()
				return err
			}

			privKey, _ := addressKey.ECPrivKey()
			addressKey.Zero()

			privKeyBytes := privKey.Serialize()
			privKeyEncrypted, err := m.cryptoKeyPriv.Encrypt(privKeyBytes)
			privKey.Zero()
			if err != nil {
				m.lock()
				str := fmt.Sprintf("failed to encrypt private key for address %s",
					info.managedAddr.Address())
				return managerError(ErrCrypto, str, err)
			}

			switch a := info.managedAddr.(type) {
			case *managedAddress:
				a.privKeyEncrypted = privKeyEncrypted
				a.privKeyCT = privKeyBytes
			case *scriptAddress:
			}

			manager.deriveOnUnlock[0] = nil
			manager.deriveOnUnlock = manager.deriveOnUnlock[1:]
		}
	}

	m.locked = false
	saltedPassphrase := append(m.privPassphraseSalt[:], passphrase...)
	m.hashedPrivPassphrase = sha512.Sum512(saltedPassphrase)
	zero.Bytes(saltedPassphrase)
	return nil
}

func (m *Manager) NewScopedKeyManager(ns walletdb.ReadWriteBucket,
	scope KeyScope, addrSchema ScopeAddrSchema) (*ScopedKeyManager, error) {

	m.mtx.Lock()
	defer m.mtx.Unlock()

	var rootPriv *hdkeychain.ExtendedKey
	if !m.watchingOnly {

	}

	scopeBucket := ns.NestedReadWriteBucket(scopeBucketName)

	if err := createScopedManagerNS(scopeBucket, &scope); err != nil {
		return nil, err
	}

	scopeSchemas := ns.NestedReadWriteBucket(scopeSchemaBucketName)
	if scopeSchemas == nil {
		str := "scope schema bucket not found"
		return nil, managerError(ErrDatabase, str, nil)
	}
	scopeKey := scopeToBytes(&scope)
	schemaBytes := scopeSchemaToBytes(&addrSchema)
	err := scopeSchemas.Put(scopeKey[:], schemaBytes)
	if err != nil {
		return nil, err
	}

	if !m.watchingOnly {
		err = createManagerKeyScope(
			ns, scope, rootPriv, m.cryptoKeyPub, m.cryptoKeyPriv)
		if err != nil {
			return nil, err
		}
	}

	m.scopedManagers[scope] = &ScopedKeyManager{
		scope:       scope,
		addrSchema:  addrSchema,
		rootManager: m,
		addrs:       make(map[addrKey]ManagedAddress),
		acctInfo:    make(map[uint32]*accountInfo),
		privKeyCache: lru.NewCache[DerivationPath, *cachedKey](
			defaultPrivKeyCacheSize,
		),
	}

	m.externalAddrSchemas[addrSchema.ExternalAddrType] = append(
		m.externalAddrSchemas[addrSchema.ExternalAddrType], scope)
	m.internalAddrSchemas[addrSchema.InternalAddrType] = append(
		m.internalAddrSchemas[addrSchema.InternalAddrType], scope)

	return m.scopedManagers[scope], nil
}

func managerExists(ns walletdb.ReadBucket) bool {
	if ns == nil {
		return false
	}
	mainBucket := ns.NestedReadBucket(mainBucketName)
	return mainBucket != nil
}

func (m *Manager) FetchScopedKeyManager(scope KeyScope) (*ScopedKeyManager, error) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	sm, ok := m.scopedManagers[scope]
	if !ok {
		str := fmt.Sprintf("scope `%v` not found", scope)
		return nil, managerError(ErrScopeNotFound, str, nil)
	}

	return sm, nil
}

func loadManager(ns walletdb.ReadBucket, pubPassphrase []byte,
	chainParams *chaincfg.Params) (*Manager, error) {

	version, err := fetchManagerVersion(ns)
	if err != nil {
		str := "failed to fetch manager version"
		return nil, managerError(ErrDatabase, str, err)
	}

	if version < latestMgrVersion {
		str := "database upgrade required"
		return nil, managerError(ErrUpgrade, str, nil)
	} else if version > latestMgrVersion {
		str := "database version is greater than latest understood version"
		return nil, managerError(ErrUpgrade, str, nil)
	}

	watchingOnly, err := fetchWatchingOnly(ns)
	if err != nil {
		return nil, maybeConvertDbError(err)
	}

	masterKeyPubParams, masterKeyPrivParams, err := fetchMasterKeyParams(ns)
	if err != nil {
		return nil, maybeConvertDbError(err)
	}

	cryptoKeyPubEnc, cryptoKeyPrivEnc, cryptoKeyScriptEnc, err := fetchCryptoKeys(ns)
	if err != nil {
		return nil, maybeConvertDbError(err)
	}

	birthday, err := fetchBirthday(ns)
	if err != nil {
		return nil, maybeConvertDbError(err)
	}

	var masterKeyPriv snacl.SecretKey
	if !watchingOnly {
		err := masterKeyPriv.Unmarshal(masterKeyPrivParams)
		if err != nil {
			str := "failed to unmarshal master private key"
			return nil, managerError(ErrCrypto, str, err)
		}
	}
	fmt.Println("反序列化 Master Priv Key 参数")

	var masterKeyPub snacl.SecretKey
	if err := masterKeyPub.Unmarshal(masterKeyPubParams); err != nil {
		str := "failed to unmarshal master public key"
		return nil, managerError(ErrCrypto, str, err)
	}
	fmt.Println("反序列化 Master Pub Key 参数")

	if err := masterKeyPub.DeriveKey(&pubPassphrase); err != nil {
		str := "invalid passphrase for master public key"
		return nil, managerError(ErrWrongPassphrase, str, err)
	}
	fmt.Println("根据 `pubPassphrase` 派生出 master pub key 的原始对象")

	cryptoKeyPub := &cryptoKey{snacl.CryptoKey{}}
	cryptoKeyPubCT, err := masterKeyPub.Decrypt(cryptoKeyPubEnc)
	if err != nil {
		str := "failed to decrypt crypto public key"
		return nil, managerError(ErrCrypto, str, err)
	}
	cryptoKeyPub.CopyBytes(cryptoKeyPubCT)
	zero.Bytes(cryptoKeyPubCT)
	fmt.Println("Decrypt Crypto Pub Key")

	var privPassphraseSalt [saltSize]byte
	_, err = rand.Read(privPassphraseSalt[:])
	if err != nil {
		str := "failed to read random source for passphrase salt"
		return nil, managerError(ErrCrypto, str, err)
	}

	scopedManagers := make(map[KeyScope]*ScopedKeyManager)
	err = forEachKeyScope(ns, func(scope KeyScope) error {
		scopeSchema, err := fetchScopeAddrSchema(ns, &scope)
		if err != nil {
			return err
		}

		scopedManagers[scope] = &ScopedKeyManager{
			scope:      scope,
			addrSchema: *scopeSchema,
			addrs:      make(map[addrKey]ManagedAddress),
			acctInfo:   make(map[uint32]*accountInfo),
			privKeyCache: lru.NewCache[DerivationPath, *cachedKey](
				defaultPrivKeyCacheSize,
			),
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	mgr := newManager(
		chainParams, &masterKeyPub, &masterKeyPriv,
		cryptoKeyPub, cryptoKeyPrivEnc, cryptoKeyScriptEnc,
		birthday, privPassphraseSalt, scopedManagers, watchingOnly)

	for _, scopedManager := range scopedManagers {
		scopedManager.rootManager = mgr
	}

	return mgr, nil
}

func newManager(chainParams *chaincfg.Params, masterKeyPub *snacl.SecretKey,
	masterKeyPriv *snacl.SecretKey, cryptoKeyPub EncryptorDecryptor,
	cryptoKeyPrivEncrypted, cryptoKeyScriptEncrypted []byte,
	birthday time.Time, privPassphraseSalt [saltSize]byte,
	scopedKeyManagers map[KeyScope]*ScopedKeyManager, watchingOnly bool) *Manager {

	m := &Manager{
		locked:                   true,
		masterKeyPub:             masterKeyPub,
		masterKeyPriv:            masterKeyPriv,
		cryptoKeyPub:             cryptoKeyPub,
		cryptoKeyPrivEncrypted:   cryptoKeyPrivEncrypted,
		cryptoKeyPriv:            &cryptoKey{},
		cryptoKeyScriptEncrypted: cryptoKeyScriptEncrypted,
		cryptoKeyScript:          &cryptoKey{},
		privPassphraseSalt:       privPassphraseSalt,
		scopedManagers:           scopedKeyManagers,
		externalAddrSchemas:      make(map[AddressType][]KeyScope),
		internalAddrSchemas:      make(map[AddressType][]KeyScope),
		watchingOnly:             watchingOnly,
	}

	for _, sMgr := range m.scopedManagers {
		externalType := sMgr.AddrSchema().ExternalAddrType
		internalType := sMgr.AddrSchema().InternalAddrType
		scope := sMgr.Scope()

		m.externalAddrSchemas[externalType] = append(
			m.externalAddrSchemas[externalType], scope)
		m.internalAddrSchemas[internalType] = append(
			m.internalAddrSchemas[internalType], scope)
	}

	return m
}

func newSecretKey(passphrase *[]byte, config *ScryptOptions) (*snacl.SecretKey, error) {
	secretKeyGenMtx.Lock()
	defer secretKeyGenMtx.Unlock()

	return secretKeyGen(passphrase, config)
}

// createManagerKeyScope 创建 KeyScore
/*
	root -> coinType -> account
			  |			  |
 	 	    privKey 	privKey
			  |			  |
			pubKey		pubKey
*/
func createManagerKeyScope(ns walletdb.ReadWriteBucket,
	scope KeyScope, root *hdkeychain.ExtendedKey,
	cryptoKeyPub, cryptoKeyPriv EncryptorDecryptor) error {

	coinTypeKeyPriv, err := deriveCoinTypeKey(root, scope)
	if err != nil {
		str := "failed to derive cointype extended key"
		return managerError(ErrKeyChain, str, err)
	}
	defer coinTypeKeyPriv.Zero()

	acctKeyPriv, err := deriveAccountKey(coinTypeKeyPriv, 0)
	if err != nil {
		if err == hdkeychain.ErrInvalidChild {
			str := "the provided seed is invalid"
			return managerError(ErrKeyChain, str, hdkeychain.ErrUnusableSeed)
		}
		return err
	}

	if err := checkBranchKeys(acctKeyPriv); err != nil {
		if err == hdkeychain.ErrInvalidChild {
			str := "the provided seed is unusable"
			return managerError(ErrKeyChain, str, hdkeychain.ErrUnusableSeed)
		}
		return err
	}

	acctKeyPub, err := acctKeyPriv.Neuter()
	if err != nil {
		str := "failed to convert private key for account 0"
		return managerError(ErrKeyChain, str, err)
	}

	coinTypeKeyPub, err := coinTypeKeyPriv.Neuter()
	if err != nil {
		str := "failed to convert cointype private key"
		return managerError(ErrKeyChain, str, err)
	}
	coinTypePubEnc, err := cryptoKeyPub.Encrypt([]byte(coinTypeKeyPub.String()))
	if err != nil {
		str := "failed to encrypt coin_type public key"
		return managerError(ErrCrypto, str, err)
	}
	coinTypePrivEnc, err := cryptoKeyPriv.Encrypt([]byte(coinTypeKeyPriv.String()))
	if err != nil {
		str := "failed to encrypt coin_type private key"
		return managerError(ErrCrypto, str, err)
	}

	acctPubEnc, err := cryptoKeyPub.Encrypt([]byte(acctKeyPub.String()))
	if err != nil {
		str := "failed to encrypt public key for account 0"
		return managerError(ErrCrypto, str, err)
	}
	acctPrivEnc, err := cryptoKeyPriv.Encrypt([]byte(acctKeyPriv.String()))
	if err != nil {
		str := "failed to encrypt private key for account 0"
		return managerError(ErrCrypto, str, err)
	}

	err = putCoinTypeKeys(ns, &scope, coinTypePubEnc, coinTypePrivEnc)
	if err != nil {
		return err
	}

	err = putDefaultAccountInfo(ns, &scope,
		DefaultAccountNum, acctPubEnc, acctPrivEnc,
		0, 0, defaultAccountName)
	if err != nil {
		return err
	}

	return putDefaultAccountInfo(ns, &scope,
		ImportedAddrAccount, nil, nil,
		0, 0, ImportedAddrAccountName)
}

func deriveCoinTypeKey(masterNode *hdkeychain.ExtendedKey,
	scope KeyScope) (*hdkeychain.ExtendedKey, error) {

	if scope.Coin > maxCoinType {
		err := managerError(ErrCoinTypeTooHigh, errCoinTypeTooHigh, nil)
		return nil, err
	}

	purpose, err := masterNode.DeriveNonStandard(
		scope.Purpose + hdkeychain.HardenedKeyStart)
	fmt.Printf("【 derive purpose key 】=> %v \n", purpose)
	if err != nil {
		return nil, err
	}

	coinTypeKey, err := purpose.DeriveNonStandard(
		scope.Coin + hdkeychain.HardenedKeyStart)
	fmt.Printf("【 derive coin_type key 】=> %v \n", coinTypeKey)
	if err != nil {
		return nil, err
	}

	return coinTypeKey, nil
}

func deriveAccountKey(coinTypeKey *hdkeychain.ExtendedKey,
	account uint32) (*hdkeychain.ExtendedKey, error) {

	if account > MaxAccountNum {
		err := managerError(ErrAccountNumTooHigh, errAcctTooHigh, nil)
		return nil, err
	}

	acctKey, err := coinTypeKey.DeriveNonStandard(
		account + hdkeychain.HardenedKeyStart)
	fmt.Printf("【 derive account key 】=> %v \n", acctKey)
	return acctKey, err
}

func checkBranchKeys(acctKey *hdkeychain.ExtendedKey) error {
	if _, err := acctKey.DeriveNonStandard(ExternalBranch); err != nil {
		return err
	}
	_, err := acctKey.DeriveNonStandard(InternalBranch)
	return err
}

func ValidateAccountName(name string) error {
	if name == "" {
		str := "accounts may not be named the empty string"
		return managerError(ErrInvalidAccount, str, nil)
	}
	if isReservedAccountName(name) {
		str := "reserved account name"
		return managerError(ErrInvalidAccount, str, nil)
	}
	return nil
}

func isReservedAccountName(name string) bool {
	return name == ImportedAddrAccountName
}

type EncryptorDecryptor interface {
	Encrypt(in []byte) ([]byte, error)
	Decrypt(in []byte) ([]byte, error)
	Bytes() []byte
	CopyBytes([]byte)
	Zero()
}
