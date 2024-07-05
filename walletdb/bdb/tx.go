package bdb

import (
	"github.com/czh0526/btc-wallet/walletdb"
	"go.etcd.io/bbolt"
)

type transaction struct {
	boltTx *bbolt.Tx
}

func (tx *transaction) ForEachBucket(fn func(key []byte) error) error {
	return convertErr(tx.boltTx.ForEach(
		func(name []byte, _ *bbolt.Bucket) error {
			return fn(name)
		}),
	)
}

func (tx *transaction) CreateTopLevelBucket(key []byte) (walletdb.ReadWriteBucket, error) {
	boltBucket, err := tx.boltTx.CreateBucketIfNotExists(key)
	if err != nil {
		return nil, convertErr(err)
	}
	return &bucket{
		Bucket: boltBucket,
		name:   key,
	}, nil
}

func (tx *transaction) DeleteTopLevelBucket(key []byte) error {
	err := tx.boltTx.DeleteBucket(key)
	if err != nil {
		return convertErr(err)
	}
	return nil
}

func (tx *transaction) ReadBucket(key []byte) walletdb.ReadBucket {
	return tx.ReadWriteBucket(key)
}

func (tx *transaction) ReadWriteBucket(key []byte) walletdb.ReadWriteBucket {
	boltBucket := tx.boltTx.Bucket(key)
	if boltBucket == nil {
		return nil
	}
	return &bucket{
		Bucket: boltBucket,
		name:   key,
	}
}

func (tx *transaction) Commit() error {
	return convertErr(tx.boltTx.Commit())
}

func (tx *transaction) Rollback() error {
	return convertErr(tx.boltTx.Rollback())
}

func (tx *transaction) OnCommit(f func()) {
	tx.boltTx.OnCommit(f)
}

var _ walletdb.ReadWriteTx = (*transaction)(nil)
