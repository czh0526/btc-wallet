package waddrmgr

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/txscript"
	"github.com/czh0526/btc-wallet/internal/zero"
	"github.com/czh0526/btc-wallet/walletdb"
	"sync"
)

// 包装一个 btc 地址，以及其附带的公私钥数据
type managedAddress struct {
	manager          *ScopedKeyManager
	derivationPath   DerivationPath
	address          btcutil.Address
	imported         bool
	internal         bool
	compressed       bool
	addrType         AddressType
	pubKey           *btcec.PublicKey
	privKeyEncrypted []byte
	privKeyCT        []byte
	privKeyMutex     sync.Mutex
}

func (a *managedAddress) InternalAccount() uint32 {
	return a.derivationPath.InternalAccount
}

func (a *managedAddress) Address() btcutil.Address {
	return a.address
}

func (a *managedAddress) AddrHash() []byte {
	var hash []byte

	switch n := a.address.(type) {
	case *btcutil.AddressPubKeyHash:
		hash = n.Hash160()[:]
	case *btcutil.AddressScriptHash:
		hash = n.Hash160()[:]
	case *btcutil.AddressWitnessPubKeyHash:
		hash = n.Hash160()[:]
	case *btcutil.AddressTaproot:
		hash = n.WitnessProgram()
	}

	return hash
}

func (a *managedAddress) Imported() bool {
	return a.imported
}

func (a *managedAddress) Internal() bool {
	return a.internal
}

func (a *managedAddress) Compressed() bool {
	return a.compressed
}

func (a *managedAddress) Used(ns walletdb.ReadBucket) bool {
	return a.manager.fetchUsed(ns, a.AddrHash())
}

func (a *managedAddress) AddrType() AddressType {
	return a.addrType
}

func (a *managedAddress) PubKey() *btcec.PublicKey {
	return a.pubKey
}

func (a *managedAddress) ExportPubKey() string {
	return hex.EncodeToString(a.pubKeyBytes())
}

func (a *managedAddress) PrivKey() (*btcec.PrivateKey, error) {
	if a.manager.rootManager.WatchOnly() {
		return nil, managerError(ErrWatchingOnly, errWatchingOnly, nil)
	}

	a.manager.mtx.Lock()
	defer a.manager.mtx.Unlock()

	if a.manager.rootManager.IsLocked() {
		return nil, managerError(ErrLocked, errLocked, nil)
	}

	privKeyCopy, err := a.unlock(a.manager.rootManager.cryptoKeyPriv)
	if err != nil {
		return nil, err
	}

	privKey, _ := btcec.PrivKeyFromBytes(privKeyCopy)
	zero.Bytes(privKeyCopy)
	return privKey, nil
}

func (a *managedAddress) ExportPrivKey() (*btcutil.WIF, error) {
	pk, err := a.PrivKey()
	if err != nil {
		return nil, err
	}

	return btcutil.NewWIF(pk, a.manager.rootManager.chainParams, a.compressed)
}

func (a *managedAddress) DerivationInfo() (KeyScope, DerivationPath, bool) {
	var (
		scope KeyScope
		path  DerivationPath
	)

	if a.imported {
		return scope, path, false
	}

	return a.manager.Scope(), a.derivationPath, true
}

func (a *managedAddress) unlock(key EncryptorDecryptor) ([]byte, error) {
	a.privKeyMutex.Lock()
	defer a.privKeyMutex.Unlock()

	if len(a.privKeyEncrypted) == 0 {
		return nil, managerError(ErrWatchingOnly, errWatchingOnly, nil)
	}

	if len(a.privKeyCT) == 0 {
		privKey, err := key.Decrypt(a.privKeyEncrypted)
		if err != nil {
			str := fmt.Sprintf("failed to decrypt private key for %s", a.address)
			return nil, managerError(ErrCrypto, str, err)
		}

		a.privKeyCT = privKey
	}

	privKeyCopy := make([]byte, len(a.privKeyCT))
	copy(privKeyCopy, a.privKeyCT)
	return privKeyCopy, nil
}

func (a *managedAddress) lock() {
	a.privKeyMutex.Lock()
	defer a.privKeyMutex.Unlock()

	zero.Bytes(a.privKeyCT)
	a.privKeyCT = nil
}

func (a *managedAddress) pubKeyBytes() []byte {
	if a.addrType == TaprootPubKey {
		return schnorr.SerializePubKey(a.pubKey)
	}
	if a.compressed {
		return a.pubKey.SerializeCompressed()
	}
	return a.pubKey.SerializeUncompressed()
}

func (a *managedAddress) Validate(msg [32]byte, priv *btcec.PrivateKey) error {
	basePubKey := priv.PubKey()
	if !a.pubKey.IsEqual(basePubKey) {
		return fmt.Errorf("%w: expected %x, got %x", ErrPubKeyMismatch,
			basePubKey.SerializeUncompressed(),
			a.pubKey.SerializeUncompressed())
	}

	addr, err := newManagedAddressWithoutPrivKey(
		a.manager, a.derivationPath, a.pubKey, a.compressed, a.addrType)
	if err != nil {
		return fmt.Errorf("unable to re-create addr: %v", err)
	}
	if addr.address.String() != a.address.String() {
		return fmt.Errorf("%w: expected %x, got %x", ErrAddrMismatch,
			addr.address.String(), a.address.String())
	}

	var sig signature
	addrPrivKey, _ := btcec.PrivKeyFromBytes(a.privKeyCT)
	switch a.addrType {
	case NestedWitnessPubKey, PubKeyHash, WitnessPubKey:
		sig = ecdsa.Sign(addrPrivKey, msg[:])
	case TaprootPubKey:
		sig, err = schnorr.Sign(addrPrivKey, msg[:])
		if err != nil {
			return fmt.Errorf("unable to generate validate schnorr sig: %w", err)
		}
	default:
		return fmt.Errorf("unable to validate addr, unknown type: %v", a.addrType)
	}

	if !sig.Verify(msg[:], basePubKey) {
		return ErrInvalidSignature
	}

	return nil
}

// 根据 ExtendedKey 构建一个地址对象
func newManagedAddressFromExtKey(s *ScopedKeyManager,
	derivationPath DerivationPath, key *hdkeychain.ExtendedKey,
	addrType AddressType, acctInfo *accountInfo) (*managedAddress, error) {

	var managedAddr *managedAddress
	if key.IsPrivate() {
		privKey, err := key.ECPrivKey()
		if err != nil {
			return nil, err
		}

		managedAddr, err = newManagedAddress(
			s, derivationPath, privKey, true, addrType, acctInfo)
		if err != nil {
			return nil, err
		}
	} else {
		pubKey, err := key.ECPubKey()
		if err != nil {
			return nil, err
		}

		managedAddr, err = newManagedAddressWithoutPrivKey(
			s, derivationPath, pubKey, true, addrType)
		if err != nil {
			return nil, err
		}
	}

	return managedAddr, nil
}

func newManagedAddress(s *ScopedKeyManager, derivationPath DerivationPath,
	privKey *btcec.PrivateKey, compressed bool,
	addrType AddressType, acctInfo *accountInfo) (*managedAddress, error) {

	// 私钥数据
	privKeyBytes := privKey.Serialize()
	privKeyEncrypted, err := s.rootManager.cryptoKeyPriv.Encrypt(privKeyBytes)
	if err != nil {
		str := "failed to encrypt private key"
		return nil, managerError(ErrCrypto, str, err)
	}

	// 根据公钥生成地址
	ecPubKey := privKey.PubKey()
	managedAddr, err := newManagedAddressWithoutPrivKey(
		s, derivationPath, ecPubKey, true, addrType)
	if err != nil {
		return nil, err
	}
	// 设置地址的私钥属性
	managedAddr.privKeyEncrypted = privKeyEncrypted
	managedAddr.privKeyCT = privKeyBytes

	// 校验地址的功能
	var msg [32]byte
	if _, err := rand.Read(msg[:]); err != nil {
		return nil, fmt.Errorf("unable to read random challenge for addr validation: %w", err)
	}

	err = managedAddr.Validate(msg, privKey)
	if err != nil {
		return nil, fmt.Errorf("addr validation for addr=%v failed: %w", managedAddr.address, err)
	}

	if acctInfo == nil || acctInfo.acctKeyPriv == nil {
		return managedAddr, nil
	}

	// 再次校验
	rederivedKey, err := s.deriveKey(
		acctInfo, managedAddr.derivationPath.Branch,
		managedAddr.derivationPath.Index, true)
	if err != nil {
		return nil, fmt.Errorf("unable to re-derive key: %w", err)
	}

	freshPrivKey, err := rederivedKey.ECPrivKey()
	if err != nil {
		return nil, fmt.Errorf("unable to re-derive private key: %w", err)
	}
	err = managedAddr.Validate(msg, freshPrivKey)
	if err != nil {
		return nil, fmt.Errorf("addr validation for addr=%v failed "+
			"after rederiving: %w", managedAddr.address, err)
	}

	return managedAddr, nil
}

func newManagedAddressWithoutPrivKey(m *ScopedKeyManager,
	derivationPath DerivationPath, pubKey *btcec.PublicKey,
	compressed bool, addrType AddressType) (*managedAddress, error) {

	var pubKeyHash []byte
	if compressed {
		pubKeyHash = btcutil.Hash160(pubKey.SerializeCompressed())
	} else {
		pubKeyHash = btcutil.Hash160(pubKey.SerializeUncompressed())
	}

	var address btcutil.Address
	var err error

	switch addrType {
	case NestedWitnessPubKey:
		witAddr, err := btcutil.NewAddressWitnessPubKeyHash(
			pubKeyHash, m.rootManager.chainParams)
		if err != nil {
			return nil, err
		}

		witnessProgram, err := txscript.PayToAddrScript(witAddr)
		if err != nil {
			return nil, err
		}

		address, err = btcutil.NewAddressScriptHash(
			witnessProgram, m.rootManager.chainParams)
		if err != nil {
			return nil, err
		}

	case PubKeyHash:
		address, err = btcutil.NewAddressPubKeyHash(
			pubKeyHash, m.rootManager.chainParams)
		if err != nil {
			return nil, err
		}
	case WitnessPubKey:
		address, err = btcutil.NewAddressWitnessPubKeyHash(
			pubKeyHash, m.rootManager.chainParams)
		if err != nil {
			return nil, err
		}
	case TaprootPubKey:
		tapKey := txscript.ComputeTaprootKeyNoScript(pubKey)
		address, err = btcutil.NewAddressTaproot(
			schnorr.SerializePubKey(tapKey), m.rootManager.chainParams)
		if err != nil {
			return nil, err
		}
	}

	return &managedAddress{
		manager:          m,
		address:          address,
		derivationPath:   derivationPath,
		imported:         false,
		internal:         false,
		addrType:         addrType,
		compressed:       compressed,
		pubKey:           pubKey,
		privKeyEncrypted: nil,
		privKeyCT:        nil,
	}, nil
}
