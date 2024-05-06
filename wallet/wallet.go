package wallet

import (
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/czh0526/btc-wallet/waddrmgr"
	"github.com/czh0526/btc-wallet/walletdb"
	"github.com/czh0526/btc-wallet/wtxmgr"
	"sync"
	"time"
)

const (
	defaultSyncRetryInterval = 5 * time.Second
)

var (
	waddrmgrNamespaceKey = []byte("waddrmgr")
	wtxmgrNamespaceKey   = []byte("wtxmgr")
)

type Wallet struct {
	db      walletdb.DB
	Manager *waddrmgr.Manager
	TxStore *wtxmgr.Store

	started bool
	quit    chan struct{}
	quitMu  sync.Mutex

	wg sync.WaitGroup
}

func (w *Wallet) Start() {
	fmt.Printf("Wallet::Start() was running ... \n")
	w.quitMu.Lock()

	select {
	case <-w.quit:
		w.WaitForShutdown()
		w.quit = make(chan struct{})
	default:
		if w.started {
			w.quitMu.Unlock()
			return
		}
		w.started = true
	}
	w.quitMu.Unlock()

	w.wg.Add(2)
	go w.txCreator()
	go w.walletLocker()
}

func (w *Wallet) Stop() {
	quit := w.quitChan()

	select {
	case <-quit:
	default:
		fmt.Printf("Wallet::Stop() send quit signal to chan \n")
		close(quit)
	}
}

func (w *Wallet) WaitForShutdown() {
	w.wg.Wait()
}

func (w *Wallet) txCreator() {
	quit := w.quitChan()
out:
	for {
		select {
		case <-quit:
			break out
		}
	}
	w.wg.Done()
	fmt.Println("txCreator finished")
}

func (w *Wallet) walletLocker() {
	var timeout <-chan time.Time
	quit := w.quitChan()
out:
	for {
		select {
		case <-quit:
			break out
		case <-timeout:
		}
	}
	w.wg.Done()
	fmt.Println("walletLocker finished")
}

func (w *Wallet) quitChan() chan struct{} {
	w.quitMu.Lock()
	c := w.quit
	w.quitMu.Unlock()
	return c
}

func CreateWithCallback(db walletdb.DB, pubPass, privPass []byte,
	rootKey *hdkeychain.ExtendedKey, params *chaincfg.Params,
	birthday time.Time, cb func(walletdb.ReadWriteTx) error) error {

	return create(
		db, pubPass, privPass, rootKey, params, birthday, false, cb)
}

func CreateWatchingOnlyWithCallback(db walletdb.DB, pubPass []byte,
	params *chaincfg.Params, birthday time.Time, cb func(walletdb.ReadWriteTx) error) error {

	return create(
		db, pubPass, nil, nil, params, birthday, true, cb)
}

func create(db walletdb.DB, pubPass, privPass []byte,
	rootKey *hdkeychain.ExtendedKey, params *chaincfg.Params,
	birthday time.Time, isWatchingOnly bool, cb func(walletdb.ReadWriteTx) error) error {

	if !isWatchingOnly && rootKey == nil {
		fmt.Println("【 Generate Seed 】")
		hdSeed, err := hdkeychain.GenerateSeed(
			hdkeychain.RecommendedSeedLen)
		if err != nil {
			return err
		}

		fmt.Println("【 New Master Key 】")
		rootKey, err = hdkeychain.NewMaster(hdSeed, params)
		if err != nil {
			return fmt.Errorf("failed to derive master extended key")
		}
	}

	if !isWatchingOnly && rootKey != nil && !rootKey.IsPrivate() {
		return fmt.Errorf("need extended private key for wallet that is not watching only")
	}

	fmt.Println("【 Create Wallet 】")
	return walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		addrmgrNs, err := tx.CreateTopLevelBucket(waddrmgrNamespaceKey)
		if err != nil {
			return err
		}
		fmt.Printf("\t=> %s \n", waddrmgrNamespaceKey)

		txmgrNs, err := tx.CreateTopLevelBucket(wtxmgrNamespaceKey)
		if err != nil {
			return err
		}
		fmt.Printf("\t=> %s \n", wtxmgrNamespaceKey)

		err = waddrmgr.Create(
			addrmgrNs, rootKey, pubPass, privPass, params, nil, birthday)
		if err != nil {
			return err
		}

		err = wtxmgr.Create(txmgrNs)
		if err != nil {
			return err
		}

		if cb != nil {
			return cb(tx)
		}

		return nil
	})
}

func OpenWithRetry(db walletdb.DB, pubPass []byte,
	params *chaincfg.Params, recoveryWindow uint32,
	syncRetryInterval time.Duration) (*Wallet, error) {

	var (
		addrMgr *waddrmgr.Manager
		txMgr   *wtxmgr.Store
	)
	fmt.Println("【 Open wallet 】")

	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		var err error
		addrMgrBucket := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		if addrMgrBucket == nil {
			return errors.New("missing address manager namespace")
		}

		txMgrBucket := tx.ReadWriteBucket(wtxmgrNamespaceKey)
		if txMgrBucket == nil {
			return errors.New("missing transaction manager namespace")
		}

		addrMgr, err = waddrmgr.Open(addrMgrBucket, pubPass, params)
		if err != nil {
			return err
		}
		txMgr, err = wtxmgr.Open(txMgrBucket, params)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	w := &Wallet{
		db:      db,
		Manager: addrMgr,
		TxStore: txMgr,
		quit:    make(chan struct{}),
	}

	return w, nil
}
