package waddrmgr

import (
	"bytes"
	"fmt"
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

func TestNewRawAccount(t *testing.T) {
	t.Parallel()

	teardown, db := emptyDB(t)
	defer teardown()

	var mgr *Manager
	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns, err := tx.CreateTopLevelBucket(waddrmgrNamespaceKey)
		if err != nil {
			return err
		}

		err = Create(ns, rootKey, pubPassphrase, privPassphrase,
			&chaincfg.MainNetParams, &FastScryptOptions, time.Time{})
		if err != nil {
			return err
		}

		mgr, err = Open(ns, pubPassphrase, &chaincfg.MainNetParams)
		if err != nil {
			return err
		}

		return mgr.Unlock(ns, privPassphrase)
	})
	if err != nil {
		t.Fatalf("create/open: unexpected error: %v", err)
	}
	defer mgr.Close()

	scopedMgr, err := mgr.FetchScopedKeyManager(KeyScopeBIP0084)
	if err != nil {
		t.Fatalf("fetch scope %v: %v", KeyScopeBIP0084, err)
	}

	const accountNum = 1000
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		return scopedMgr.NewRawAccount(ns, accountNum)
	})
	if err != nil {
		t.Fatalf("unable to create new account: %v", err)
	}

	testNewRawAccount(t, mgr, db, accountNum, scopedMgr)
}

func testNewRawAccount(t *testing.T, _ *Manager, db walletdb.DB, accountNum uint32, scopedMgr *ScopedKeyManager) {
	var accountAddrNext ManagedAddress
	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)

		addrs, err := scopedMgr.NextExternalAddresses(
			ns, accountNum, 1)
		if err != nil {
			return err
		}

		accountAddrNext = addrs[0]
		return nil
	})
	if err != nil {
		t.Fatalf("unable to create addr: %v", err)
	}

	var accountTargetAddr ManagedAddress
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)

		keyPath := DerivationPath{
			InternalAccount: accountNum,
			Account:         hdkeychain.HardenedKeyStart,
			Branch:          0,
			Index:           0,
		}
		accountTargetAddr, err = scopedMgr.DeriveFromKeyPath(ns, keyPath)
		return err
	})
	if err != nil {
		t.Fatalf("unable to derive addr: %v", err)
	}

	if accountAddrNext.AddrType() != accountTargetAddr.AddrType() {
		t.Fatalf("wrong addr type: %v vs %v", accountAddrNext.AddrType(), accountTargetAddr.AddrType())
	}
	if !bytes.Equal(accountAddrNext.AddrHash(), accountTargetAddr.AddrHash()) {
		t.Fatalf("wrong pubkey hash: %x vs %x", accountAddrNext.AddrHash(), accountTargetAddr.AddrHash())
	}
}

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

type testContext struct {
	t               *testing.T
	caseName        string
	db              walletdb.DB
	rootManager     *Manager
	manager         *ScopedKeyManager
	internalAccount uint32
	create          bool
	unlocked        bool
	watchingOnly    bool
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

		fmt.Println("\nCreate Manager => ")
		err = Create(
			ns, caseKey, pubPassphrase, casePrivPassphrase,
			&chaincfg.MainNetParams, &FastScryptOptions, time.Time{})
		if err != nil {
			return err
		}

		fmt.Println("\nOpen Manager => ")
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
		t.Errorf("(%s) Create/Open: unexpected error: %v", caseName, err)
		return
	}

	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		return Create(
			ns, caseKey, pubPassphrase, casePrivPassphrase,
			&chaincfg.MainNetParams, &FastScryptOptions, time.Time{})
	})
	if !checkManagerError(t, fmt.Sprintf("(%s) Create existing", caseName), err, ErrAlreadyExists) {
		mgr.Close()
		return
	}

	scopedMgr, err := mgr.FetchScopedKeyManager(KeyScopeBIP0044)
	if err != nil {
		t.Fatalf("(%s) unable to fetch default scope: %v", caseName, err)
	}

	testManagerAPI(&testContext{
		t:               t,
		caseName:        caseName,
		db:              db,
		manager:         scopedMgr,
		rootManager:     mgr,
		internalAccount: 0,
		create:          true,
		watchingOnly:    caseCreatedWatchingOnly,
	}, caseCreatedWatchingOnly)
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

func testManagerAPI(tc *testContext, caseCreateWatchingOnly bool) {
	if !caseCreateWatchingOnly {
		testLocking(tc)

		tc.internalAccount = 0
		testNewAccount(tc)
	}
}

func testLocking(tc *testContext) bool {
	if tc.unlocked {
		tc.t.Error("testLocking called with an unlocked manager")
		return false
	}
	if !tc.rootManager.IsLocked() {
		tc.t.Error("IsLocked: returned false on locked manager")
		return false
	}

	err := tc.rootManager.Lock()
	wantErrCode := ErrLocked
	if tc.watchingOnly {
		wantErrCode = ErrWatchingOnly
	}
	if !checkManagerError(tc.t, "Lock", err, wantErrCode) {
		return false
	}

	err = walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		return tc.rootManager.Unlock(ns, privPassphrase)
	})
	if tc.watchingOnly {
		if !checkManagerError(tc.t, "Unlock", err, ErrWatchingOnly) {
			return false
		}
	} else if err != nil {
		tc.t.Error("Unlock: unexpected error:", err)
		return false
	}
	if !tc.watchingOnly && tc.rootManager.IsLocked() {
		tc.t.Error("IsLocked: returned false on unlocked manager")
		return false
	}

	return true
}

func testNewAccount(tc *testContext) bool {
	// 1
	if tc.watchingOnly {
		err := walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
			ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
			_, err := tc.manager.NewAccount(ns, "test")
			return err
		})
		if !checkManagerError(tc.t, "Create account in watching-only mode", err, ErrWatchingOnly) {
			tc.manager.Close()
			return false
		}
		return true
	}

	// 2
	err := walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		_, err := tc.manager.NewAccount(ns, "test")
		return err
	})
	if !checkManagerError(tc.t,
		"Create account when wallet is locked", err, ErrLocked) {
		tc.manager.Close()
		return false
	}

	// 3
	err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		err = tc.rootManager.Unlock(ns, privPassphrase)
		return err
	})
	if err != nil {
		tc.t.Errorf("Unlock: unexpected error: %v", err)
		return false
	}
	tc.unlocked = true

	// 4
	testName := "acct-create"
	expectedAccount := tc.internalAccount + 1
	if !tc.create {
		testName = "acct-open"
		expectedAccount++
	}
	var account uint32
	err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		var err error
		account, err = tc.manager.NewAccount(ns, testName)
		return err
	})
	if err != nil {
		tc.t.Errorf("NewAccount: unexpected error: %v", err)
		return false
	}
	if account != expectedAccount {
		tc.t.Errorf("NewAccount: account mismatch -- got %d, want %d", account, expectedAccount)
		return false
	}

	// 5 Test duplicate account name error
	err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		_, err := tc.manager.NewAccount(ns, testName)
		return err
	})
	wantErrCode := ErrDuplicateAccount
	if !checkManagerError(tc.t, testName, err, wantErrCode) {
		return false
	}

	return true
}
