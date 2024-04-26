package bdb

import (
	"github.com/czh0526/btc-wallet/walletdb"
	"go.etcd.io/bbolt"
)

type transaction struct {
	boltTx *bbolt.Tx
}

func (tx *transaction) ForEachBucket(f func(key []byte) error) error {
	//TODO implement me
	panic("implement me")
}

func (tx *transaction) Rollback() error {
	return convertErr(tx.boltTx.Rollback())
}

func (tx *transaction) CreateTopLevelBucket(key []byte) (walletdb.ReadWriteBucket, error) {
	boltBucket, err := tx.boltTx.CreateBucketIfNotExists(key)
	if err != nil {
		return nil, convertErr(err)
	}
	return (*bucket)(boltBucket), nil
}

func (tx *transaction) DeleteTopLevelBucket(key []byte) error {
	//TODO implement me
	panic("implement me")
}

func (tx *transaction) Commit() error {
	return convertErr(tx.boltTx.Commit())
}

func (tx *transaction) OnCommit(f func()) {
	//TODO implement me
	panic("implement me")
}

func (tx *transaction) ReadBucket(key []byte) walletdb.ReadBucket {
	return tx.ReadWriteBucket(key)
}

func (tx *transaction) ReadWriteBucket(key []byte) walletdb.ReadWriteBucket {
	boltBucket := tx.boltTx.Bucket(key)
	if boltBucket == nil {
		return nil
	}
	return (*bucket)(boltBucket)
}

var _ walletdb.ReadWriteTx = (*transaction)(nil)
