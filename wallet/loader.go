package wallet

import (
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/czh0526/btc-wallet/walletdb"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	ErrLoaded = errors.New("wallet already loaded")

	ErrNotLoaded = errors.New("wallet not loaded")

	ErrExists = errors.New("wallet already exists")
)

const (
	WalletDBName = "wallet.db"
)

type loaderConfig struct {
	walletSyncRetryInterval time.Duration
}

type Loader struct {
	cfg            *loaderConfig
	callbacks      []func(*Wallet)
	chainParams    *chaincfg.Params
	dbDirPath      string
	noFreelistSync bool
	timeout        time.Duration
	recoveryWindow uint32

	localDB bool
	wallet  *Wallet
	db      walletdb.DB

	walletExists  func() (bool, error)
	walletCreated func(db walletdb.ReadWriteTx) error

	mu sync.Mutex
}

func NewLoader(chainParams *chaincfg.Params, dbDirPath string,
	noFreelistSync bool, timeout time.Duration, recoveryWindow uint32) *Loader {

	cfg := defaultLoaderConfig()

	return &Loader{
		cfg:            cfg,
		chainParams:    chainParams,
		dbDirPath:      dbDirPath,
		noFreelistSync: noFreelistSync,
		timeout:        timeout,
		recoveryWindow: recoveryWindow,
		localDB:        true,
	}
}

func NewLoaderWithDB(chainParams *chaincfg.Params, recoveryWindow uint32,
	db walletdb.DB, walletExists func() (bool, error)) (*Loader, error) {

	if db == nil {
		return nil, fmt.Errorf("no DB provided")
	}

	if walletExists == nil {
		return nil, fmt.Errorf("unable to check if wallet exists")
	}

	cfg := defaultLoaderConfig()

	return &Loader{
		cfg:            cfg,
		chainParams:    chainParams,
		recoveryWindow: recoveryWindow,
		localDB:        false,
		walletExists:   walletExists,
		db:             db,
	}, nil
}

func (l *Loader) CreateNewWallet(pubPassphrase, privPassphrase, seed []byte,
	bday time.Time) (*Wallet, error) {

	var (
		rootKey *hdkeychain.ExtendedKey
		err     error
	)

	if seed != nil {
		if len(seed) < hdkeychain.MinSeedBytes ||
			len(seed) > hdkeychain.MaxSeedBytes {
			return nil, hdkeychain.ErrInvalidSeedLen
		}

		fmt.Println("【 Create Root Key 】")
		rootKey, err = hdkeychain.NewMaster(seed, l.chainParams)
		if err != nil {
			return nil, fmt.Errorf("failed to derive master extended key")
		}
	}

	return l.createNewWallet(
		pubPassphrase, privPassphrase, rootKey, bday, false)
}
func (l *Loader) WalletExists() (bool, error) {
	if l.localDB {
		dbPath := filepath.Join(l.dbDirPath, WalletDBName)
		return fileExists(dbPath)
	}

	return l.walletExists()
}

func (l *Loader) createNewWallet(pubPassphrase, privPassphrase []byte, rootKey *hdkeychain.ExtendedKey,
	bday time.Time, isWatchingOnly bool) (*Wallet, error) {

	defer l.mu.Unlock()
	l.mu.Lock()

	if l.wallet != nil {
		return nil, ErrLoaded
	}

	exists, err := l.WalletExists()
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrExists
	}

	if l.localDB {
		dbPath := filepath.Join(l.dbDirPath, WalletDBName)

		err = os.MkdirAll(l.dbDirPath, 0700)
		if err != nil {
			return nil, err
		}

		l.db, err = walletdb.Create(
			"bdb", dbPath, l.noFreelistSync, l.timeout)
		if err != nil {
			return nil, err
		}
	}

	if isWatchingOnly {
		err := CreateWatchingOnlyWithCallback(
			l.db, pubPassphrase, l.chainParams, bday, l.walletCreated)
		if err != nil {
			return nil, err
		}
	} else {
		err = CreateWithCallback(
			l.db, pubPassphrase, privPassphrase, rootKey,
			l.chainParams, bday, l.walletCreated)
		if err != nil {
			return nil, err
		}
	}

	w, err := OpenWithRetry(
		l.db, pubPassphrase, l.chainParams, l.recoveryWindow,
		l.cfg.walletSyncRetryInterval)
	if err != nil {
		return nil, err
	}
	w.Start()

	return w, nil
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

		dbPath := filepath.Join(l.dbDirPath, WalletDBName)
		l.db, err = walletdb.Open("bdb", dbPath, l.noFreelistSync, l.timeout)
		if err != nil {
			return nil, err
		}
	}

	w, err := OpenWithRetry(
		l.db, pubPassphrase, l.chainParams, l.recoveryWindow,
		l.cfg.walletSyncRetryInterval)
	if err != nil {
		if l.localDB {
			e := l.db.Close()
			if e != nil {
				fmt.Printf("Error closing DB: %v\n", e)
			}
		}
		return nil, err
	}
	w.Start()

	l.onLoaded(w)
	return w, nil
}

func (l *Loader) onLoaded(w *Wallet) {
	for _, fn := range l.callbacks {
		fn(w)
	}
	l.wallet = w
	l.callbacks = nil
}

func (l *Loader) UnloadWallet() error {
	defer l.mu.Unlock()
	l.mu.Lock()

	if l.wallet == nil {
		return ErrNotLoaded
	}

	l.wallet.Stop()
	l.wallet.WaitForShutdown()
	fmt.Println("wallet was shutdown")
	if l.localDB {
		err := l.db.Close()
		if err != nil {
			return err
		}
	}
	fmt.Println("wallet db was closed")

	l.wallet = nil
	l.db = nil
	return nil
}

func (l *Loader) RunAfterLoad(fn func(*Wallet)) {
	l.mu.Lock()
	if l.wallet != nil {
		w := l.wallet
		l.mu.Unlock()
		fn(w)
	} else {
		l.callbacks = append(l.callbacks, fn)
		l.mu.Unlock()
	}
}

func fileExists(filePath string) (bool, error) {
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func defaultLoaderConfig() *loaderConfig {
	return &loaderConfig{
		walletSyncRetryInterval: defaultSyncRetryInterval,
	}
}
