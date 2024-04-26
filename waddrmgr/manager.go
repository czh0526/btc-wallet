package waddrmgr

import (
	"fmt"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/czh0526/btc-wallet/walletdb"
	"sync"
	"time"
)

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
	if err := createManageNS(ns, defaultScope); err != nil {
		return maybeConvertDbError(err)
	}

	return nil
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
