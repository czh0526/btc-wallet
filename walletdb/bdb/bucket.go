package bdb

import (
	"github.com/czh0526/btc-wallet/walletdb"
	"go.etcd.io/bbolt"
)

type bucket struct {
	*bbolt.Bucket
	name []byte
}

func (b *bucket) Name() []byte {
	return b.name
}

func (b *bucket) NestedReadBucket(key []byte) walletdb.ReadBucket {
	return b.NestedReadWriteBucket(key)
}

func (b *bucket) ForEach(fn func(k []byte, v []byte) error) error {
	return convertErr(b.Bucket.ForEach(fn))
}

func (b *bucket) Get(key []byte) []byte {
	return b.Bucket.Get(key)
}

func (b *bucket) ReadCursor() walletdb.ReadCursor {
	//TODO implement me
	panic("implement me")
}

func (b *bucket) NestedReadWriteBucket(key []byte) walletdb.ReadWriteBucket {
	boltBucket := b.Bucket.Bucket(key)
	if boltBucket == nil {
		return nil
	}
	return &bucket{
		Bucket: boltBucket,
		name:   key,
	}
}

func (b *bucket) CreateBucket(key []byte) (walletdb.ReadWriteBucket, error) {
	boltBucket, err := b.Bucket.CreateBucket(key)
	if err != nil {
		return nil, convertErr(err)
	}
	return &bucket{
		Bucket: boltBucket,
		name:   key,
	}, nil
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
	return convertErr(b.Bucket.Put(key, value))
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
