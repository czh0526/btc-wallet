package bdb

import (
	"github.com/czh0526/btc-wallet/walletdb"
	"go.etcd.io/bbolt"
)

type bucket bbolt.Bucket

func (b *bucket) NestedReadBucket(key []byte) walletdb.ReadBucket {
	return b.NestedReadWriteBucket(key)
}

func (b *bucket) ForEach(f func(k []byte, v []byte) error) error {
	//TODO implement me
	panic("implement me")
}

func (b *bucket) Get(key []byte) []byte {
	//TODO implement me
	panic("implement me")
}

func (b *bucket) ReadCursor() walletdb.ReadCursor {
	//TODO implement me
	panic("implement me")
}

func (b *bucket) NestedReadWriteBucket(key []byte) walletdb.ReadWriteBucket {
	boltBucket := (*bbolt.Bucket)(b).Bucket(key)
	if boltBucket == nil {
		return nil
	}
	return (*bucket)(boltBucket)
}

func (b *bucket) CreateBucket(key []byte) (walletdb.ReadWriteBucket, error) {
	boltBucket, err := (*bbolt.Bucket)(b).CreateBucket(key)
	if err != nil {
		return nil, convertErr(err)
	}
	return (*bucket)(boltBucket), nil
}

func (b *bucket) CreateBucketIfNotExists(key []byte) (walletdb.ReadWriteBucket, error) {
	//TODO implement me
	panic("implement me")
}

func (b *bucket) DeleteNestedBucket(key []byte) error {
	//TODO implement me
	panic("implement me")
}

func (b *bucket) Put(key, value []byte) error {
	//TODO implement me
	panic("implement me")
}

func (b *bucket) Delete(key []byte) error {
	//TODO implement me
	panic("implement me")
}

func (b *bucket) ReadWriteCursor() walletdb.ReadWriteCursor {
	//TODO implement me
	panic("implement me")
}

func (b *bucket) Tx() walletdb.ReadWriteTx {
	//TODO implement me
	panic("implement me")
}

func (b *bucket) NextSequence() (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (b *bucket) SetSequence(v uint64) error {
	//TODO implement me
	panic("implement me")
}

func (b *bucket) Sequence() uint64 {
	//TODO implement me
	panic("implement me")
}

var _ walletdb.ReadWriteBucket = (*bucket)(nil)
