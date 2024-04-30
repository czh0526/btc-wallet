package bdb

import (
	"github.com/czh0526/btc-wallet/walletdb"
	"go.etcd.io/bbolt"
	"io"
	"os"
	"time"
)

type db bbolt.DB

func (db *db) BeginReadTx() (walletdb.ReadTx, error) {
	//TODO implement me
	panic("implement me")
}

func (db *db) beginTx(writable bool) (*transaction, error) {
	boltTx, err := (*bbolt.DB)(db).Begin(writable)
	if err != nil {
		return nil, convertErr(err)
	}
	return &transaction{boltTx: boltTx}, nil
}

func (db *db) BeginReadWriteTx() (walletdb.ReadWriteTx, error) {
	return db.beginTx(true)
}

func (db *db) Copy(w io.Writer) error {
	//TODO implement me
	panic("implement me")
}

func (db *db) Close() error {
	return convertErr((*bbolt.DB)(db).Close())
}

func (db *db) PrintStats() string {
	//TODO implement me
	panic("implement me")
}

func (db *db) View(f func(tx walletdb.ReadTx) error, reset func()) error {
	//TODO implement me
	panic("implement me")
}

func (db *db) Update(f func(tx walletdb.ReadWriteTx) error, reset func()) error {
	reset()

	tx, err := db.BeginReadWriteTx()
	if err != nil {
		return err
	}

	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	err = f(tx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

var _ walletdb.DB = (*db)(nil)

func openDB(dbPath string, noFreelistSync bool, create bool, timeout time.Duration) (walletdb.DB, error) {
	if !create && !fileExists(dbPath) {
		return nil, walletdb.ErrDbDoesNotExist
	}

	options := &bbolt.Options{
		NoFreelistSync: noFreelistSync,
		FreelistType:   bbolt.FreelistMapType,
		Timeout:        timeout,
	}

	boltDB, err := bbolt.Open(dbPath, 0600, options)
	return (*db)(boltDB), convertErr(err)
}

func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func convertErr(err error) error {
	switch err {
	// Database open/create errors.
	case bbolt.ErrDatabaseNotOpen:
		return walletdb.ErrDbNotOpen
	case bbolt.ErrInvalid:
		return walletdb.ErrInvalid

	// Transaction errors.
	case bbolt.ErrTxNotWritable:
		return walletdb.ErrTxNotWritable
	case bbolt.ErrTxClosed:
		return walletdb.ErrTxClosed

	// Value/bucket errors.
	case bbolt.ErrBucketNotFound:
		return walletdb.ErrBucketNotFound
	case bbolt.ErrBucketExists:
		return walletdb.ErrBucketExists
	case bbolt.ErrBucketNameRequired:
		return walletdb.ErrBucketNameRequired
	case bbolt.ErrKeyRequired:
		return walletdb.ErrKeyRequired
	case bbolt.ErrKeyTooLarge:
		return walletdb.ErrKeyTooLarge
	case bbolt.ErrValueTooLarge:
		return walletdb.ErrValueTooLarge
	case bbolt.ErrIncompatibleValue:
		return walletdb.ErrIncompatibleValue
	}

	// Return the original error if none of the above applies.
	return err
}
