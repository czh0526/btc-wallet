package wallet

import (
	"errors"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcwallet/walletdb"
	"path/filepath"
	"sync"
	"time"
)

var (
	ErrLoaded = errors.New("wallet already loaded")

	ErrNotLoaded = errors.New("wallet not loaded")

	ErrExists = errors.New("wallet already exists")
)

type Loader struct {
	chainParams    *chaincfg.Params
	dbDirPath      string
	noFreelistSync bool
	timeout        time.Duration
	recoveryWindow uint32
	localDB        bool

	wallet *Wallet
	db     walletdb.DB
	mu     sync.Mutex
}

func NewLoader(chainParams *chaincfg.Params, dbDirPath string,
	noFreelistSync bool, timeout time.Duration, recoveryWindow uint32,
) *Loader {

	return &Loader{
		chainParams:    chainParams,
		dbDirPath:      dbDirPath,
		noFreelistSync: noFreelistSync,
		timeout:        timeout,
		recoveryWindow: recoveryWindow,
		localDB:        true,
	}
}

func (l *Loader) OpenExistingWallet(pubPassphrase []byte,
	canConsolePrompt bool) (*Wallet, error) {

	defer l.mu.Unlock()
	l.mu.Lock()

	if l.wallet != nil {
		return nil, ErrLoaded
	}

	if l.localDB {
		var err error
		if err = CheckCreateDir(l.dbDirPath); err != nil {
			return nil, err
		}

		dbPath := filepath.Join(l.dbDirPath, "wallet.db")
		l.db, err = walletdb.Open("bdb", dbPath, l.noFreelistSync, l.timeout)
		if err != nil {
			return nil, err
		}
	}

}
