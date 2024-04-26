package wallet

import (
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/czh0526/btc-wallet/waddrmgr"
	"github.com/czh0526/btc-wallet/walletdb"
	"github.com/czh0526/btc-wallet/wtxmgr"
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
}

func (w *Wallet) Start() {
	fmt.Printf("Wallet::Start() was not yet implemented \n")
}

func (w *Wallet) Stop() {
	fmt.Printf("Wallet::Stop() was not yet implemented \n")
}

func (w *Wallet) WaitForShutdown() {
	fmt.Printf("Wallet::WaitForShutdown() was not yet implemented \n")
}

func CreateWithCallback(db walletdb.DB, pubPass, privPass []byte,
	rootKey *hdkeychain.ExtendedKey, params *chaincfg.Params,
	birthday time.Time, cb func(walletdb.ReadWriteTx) error) error {

	return create(db, pubPass, privPass, rootKey, params, birthday, false, cb)
}

func create(db walletdb.DB, pubPass, privPass []byte,
	rootKey *hdkeychain.ExtendedKey, params *chaincfg.Params,
	birthday time.Time, isWatchingOnly bool, cb func(walletdb.ReadWriteTx) error) error {

	if !isWatchingOnly && rootKey == nil {
		hdSeed, err := hdkeychain.GenerateSeed(
			hdkeychain.RecommendedSeedLen)
		if err != nil {
			return err
		}

		rootKey, err = hdkeychain.NewMaster(hdSeed, params)
		if err != nil {
			return fmt.Errorf("failed to derive master extended key")
		}
	}

	if !isWatchingOnly && rootKey != nil && !rootKey.IsPrivate() {
		return fmt.Errorf("need extended private key for wallet that is not watching only")
	}

	return walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		addrmgrNs, err := tx.CreateTopLevelBucket(waddrmgrNamespaceKey)
		if err != nil {
			return err
		}
		txmgrNs, err := tx.CreateTopLevelBucket(wtxmgrNamespaceKey)
		if err != nil {
			return err
		}

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

	fmt.Println("Opened wallet")

	w := &Wallet{
		db:      db,
		Manager: addrMgr,
		TxStore: txMgr,
	}

	return w, nil
}
