package waddrmgr

import (
	"crypto/rand"
	"fmt"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/czh0526/btc-wallet/snacl"
	"github.com/czh0526/btc-wallet/walletdb"
	"sync"
	"time"
)

const (
	DefaultAccountNum  = 0
	defaultAccountName = "default"

	maxCoinType = hdkeychain.HardenedKeyStart - 1

	MaxAccountNum = hdkeychain.HardenedKeyStart - 2

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

	scopedManagers map[KeyScope]*ScopedKeyManager
	closed         bool
}

type ScryptOptions struct {
	N, R, P int
}

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
	if err := createManagerNS(ns, defaultScope); err != nil {
		return maybeConvertDbError(err)
	}

	if config == nil {
		config = &DefaultScryptOptions
	}

	masterKeyPub, err := newSecretKey(&pubPassphrase, config)
	if err != nil {
		str := "failed to master public key"
		return managerError(ErrCrypto, str, err)
	}

	cryptoKeyPub, err := newCryptoKey()
	if err != nil {
		str := "failed to generate crypto public key"
		return managerError(ErrCrypto, str, err)
	}

	cryptoKeyPubEnc, err := masterKeyPub.Encrypt(cryptoKeyPub.Bytes())
	if err != nil {
		str := "failed to encrypt crypto public key"
		return managerError(ErrCrypto, str, err)
	}

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
		masterKeyPriv, err = newSecretKey(&privPassphrase, config)
		if err != nil {
			str := "failed to master private key"
			return managerError(ErrCrypto, str, err)
		}
		defer masterKeyPriv.Zero()

		var privPassphraseSalt [saltSize]byte
		_, err = rand.Read(privPassphraseSalt[:])
		if err != nil {
			str := "failed to read random source for passphrase salt"
			return managerError(ErrCrypto, str, err)
		}

		cryptoKeyPriv, err := newCryptoKey()
		if err != nil {
			str := "failed to generate crypto private key"
			return managerError(ErrCrypto, str, err)
		}
		defer cryptoKeyPriv.Zero()

		cryptoKeyScript, err := newCryptoKey()
		if err != nil {
			str := "failed to generate crypto script key"
			return managerError(ErrCrypto, str, err)
		}
		defer cryptoKeyScript.Zero()

		cryptoKeyPrivEnc, err = masterKeyPriv.Encrypt(cryptoKeyPriv.Bytes())
		if err != nil {
			str := "failed to encrypt crypto private key"
			return managerError(ErrCrypto, str, err)
		}

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
			err := createManagerKeyScope(
				ns, defaultScope, rootKey, cryptoKeyPub, cryptoKeyPriv)
			if err != nil {
				return maybeConvertDbError(err)
			}
		}

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

	err = putMasterKeyParams(ns, pubParams, privParams)
	if err != nil {
		return maybeConvertDbError(err)
	}

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

func Open(ns walletdb.ReadWriteBucket, pubPassphrase []byte,
	chainParams *chaincfg.Params) (*Manager, error) {

	fmt.Println("waddrmgr.Open() has not been implemented yet")
	return &Manager{}, nil
}

func (m *Manager) Close() {
	m.closed = true
}

func managerExists(ns walletdb.ReadWriteBucket) bool {
	if ns == nil {
		return false
	}
	mainBucket := ns.NestedReadBucket(mainBucketName)
	return mainBucket != nil
}

func newSecretKey(passphrase *[]byte, config *ScryptOptions) (*snacl.SecretKey, error) {
	secretKeyGenMtx.Lock()
	defer secretKeyGenMtx.Unlock()

	return secretKeyGen(passphrase, config)
}

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
		str := "failed to encrypt cointype public key"
		return managerError(ErrCrypto, str, err)
	}
	coinTypePrivEnc, err := cryptoKeyPriv.Encrypt([]byte(coinTypeKeyPriv.String()))
	if err != nil {
		str := "failed to encrypt cointype private key"
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
	if err != nil {
		return nil, err
	}

	coinTypeKey, err := purpose.DeriveNonStandard(
		scope.Coin + hdkeychain.HardenedKeyStart)
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

	return coinTypeKey.DeriveNonStandard(
		account + hdkeychain.HardenedKeyStart)
}

func checkBranchKeys(acctKey *hdkeychain.ExtendedKey) error {
	if _, err := acctKey.DeriveNonStandard(ExternalBranch); err != nil {
		return err
	}
	_, err := acctKey.DeriveNonStandard(InternalBranch)
	return err
}

type EncryptorDecryptor interface {
	Encrypt(in []byte) ([]byte, error)
	Decrypt(in []byte) ([]byte, error)
	Bytes() []byte
	CopyBytes([]byte)
	Zero()
}

type cryptoKey struct {
	snacl.CryptoKey
}

func (ck *cryptoKey) Bytes() []byte {
	return ck.CryptoKey[:]
}

func (ck *cryptoKey) CopyBytes(from []byte) {
	copy(ck.CryptoKey[:], from)
}

func defaultNewCryptoKey() (EncryptorDecryptor, error) {
	key, err := snacl.GenerateCryptoKey()
	if err != nil {
		return nil, err
	}
	return &cryptoKey{*key}, nil
}
