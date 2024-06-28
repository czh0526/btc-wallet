package bdb

import (
	"github.com/czh0526/btc-wallet/walletdb"
	_ "github.com/czh0526/btc-wallet/walletdb/bdb"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var (
	bucketName = []byte("Bucket1")
	key        = []byte("key1")
	value      = []byte("Hello Cai.Zhihong")
)

func TestDBCreateAndWrite(t *testing.T) {
	db, err := walletdb.Create("bdb", "example.db", true, 60*time.Second)
	assert.Nil(t, err)
	defer db.Close()

	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		var bucket walletdb.ReadWriteBucket
		bucket, err = tx.CreateTopLevelBucket(bucketName)
		assert.NoError(t, err)

		err = bucket.Put(key, value)
		assert.NoError(t, err)

		return nil
	})
	assert.Nil(t, err)
}

func TestDBOpenAndView(t *testing.T) {
	db, err := walletdb.Open("bdb", "example.db", true, 60*time.Second)
	assert.Nil(t, err)
	defer db.Close()

	err = walletdb.View(db, func(tx walletdb.ReadTx) error {
		var bucket walletdb.ReadBucket
		bucket = tx.ReadBucket(bucketName)
		assert.NotNil(t, bucket)

		v := bucket.Get(key)
		assert.NotNil(t, v)
		assert.Equal(t, value, v)

		return nil
	})
	assert.Nil(t, err)
}
