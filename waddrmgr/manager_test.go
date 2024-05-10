package waddrmgr

import (
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/czh0526/btc-wallet/walletdb"
	_ "github.com/czh0526/btc-wallet/walletdb/bdb"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var (
	seed = []byte{
		0x2a, 0x64, 0xdf, 0x08, 0x5e, 0xef, 0xed, 0xd8, 0xbf,
		0xdb, 0xb3, 0x31, 0x76, 0xb5, 0xba, 0x2e, 0x62, 0xe8,
		0xbe, 0x8b, 0x56, 0xc8, 0x83, 0x77, 0x95, 0x59, 0x8b,
		0xb6, 0xc4, 0x40, 0xc0, 0x64,
	}

	rootKey, _ = hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)

	pubPassphrase  = []byte("_DJr{fL4H0O}*-0\n:V1izc)(6BomK")
	privPassphrase = []byte("81lUHXnOMZ@?XXd7O9xyDIWIbXX-lj")

	defaultDBTimeout = 10 * time.Second

	waddrmgrNamespaceKey = []byte("waddrmgrNamespace")
)

func TestManager(t *testing.T) {
	tests := []struct {
		name                string
		createdWatchingOnly bool
		rootKey             *hdkeychain.ExtendedKey
		privPassphrase      []byte
	}{
		{
			name:                "created with seed",
			createdWatchingOnly: false,
			rootKey:             rootKey,
			privPassphrase:      privPassphrase,
		},
	}

	for _, test := range tests {
		testManagerCase(
			t, test.name, test.createdWatchingOnly,
			test.privPassphrase, test.rootKey)
	}
}

func testManagerCase(t *testing.T, caseName string,
	caseCreatedWatchingOnly bool, casePrivPassphrase []byte,
	caseKey *hdkeychain.ExtendedKey) {

	teardown, db := emptyDB(t)
	defer teardown()

	if !caseCreatedWatchingOnly {
		err := walletdb.View(db, func(tx walletdb.ReadTx) error {
			ns := tx.ReadBucket(waddrmgrNamespaceKey)
			_, err := Open(ns, pubPassphrase, &chaincfg.MainNetParams)
			return err
		})
		if !checkManagerError(t, "Open non-existent", err, ErrNoExist) {
			return
		}
	}

	var mgr *Manager
	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns, err := tx.CreateTopLevelBucket(waddrmgrNamespaceKey)
		if err != nil {
			return err
		}

		err = Create(
			ns, caseKey, pubPassphrase, casePrivPassphrase,
			&chaincfg.MainNetParams, &FastScryptOptions, time.Time{})
		if err != nil {
			return err
		}

		mgr, err = Open(ns, pubPassphrase, &chaincfg.MainNetParams)
		if err != nil {
			return nil
		}

		if caseCreatedWatchingOnly {
			_, err = mgr.NewScopedKeyManager(
				ns, KeyScopeBIP0044, ScopeAddrMap[KeyScopeBIP0044])
		}
		return err
	})
	if err != nil {
		t.Errorf("(%s) Failed to Create/Open wallet: %v", caseName, err)
		return
	}
}

func emptyDB(t *testing.T) (tearDownFunc func(), db walletdb.DB) {
	dirName, err := os.MkdirTemp("", "mgrtest")
	if err != nil {
		t.Fatalf("Failed to create db temp dir:  %v", err)
	}
	dbPath := filepath.Join(dirName, "mgrtest.db")
	db, err = walletdb.Create("bdb", dbPath, true, defaultDBTimeout)
	if err != nil {
		_ = os.RemoveAll(dirName)
		t.Fatalf("Failed to create db: %v", err)
	}

	tearDownFunc = func() {
		db.Close()
		_ = os.RemoveAll(dirName)
	}

	return
}
