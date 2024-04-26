package original

import (
	"github.com/btcsuite/btcwallet/walletdb"
	_ "github.com/btcsuite/btcwallet/walletdb/bdb"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var (
	dir = "/tmp"
)

func TestDBOperation(t *testing.T) {

	var db walletdb.DB
	var err error

	dir, err = os.Getwd()
	assert.Nil(t, err)
	dbPath := filepath.Join(dir, "/wallet.db")
	_, err = os.Stat(dbPath)
	if os.IsNotExist(err) {
		db, err = walletdb.Create("bdb", dbPath, true, 60*time.Second)
		assert.Nil(t, err)
	} else {
		db, err = walletdb.Open("bdb", dbPath, true, 60*time.Second)
		assert.Nil(t, err)
	}
	defer func() {
		err = db.Close()
		assert.Nil(t, err)
	}()

	bucketKey := []byte("wallet sub package")
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		bucket := tx.ReadWriteBucket(bucketKey)
		if bucket == nil {
			_, err = tx.CreateTopLevelBucket(bucketKey)
			assert.Nil(t, err)
		}
		return nil
	})
	assert.Nil(t, err)

	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		rootBucket := tx.ReadWriteBucket(bucketKey)

		key := []byte("my-key")
		value := []byte("my-value")
		err = rootBucket.Put(key, value)
		assert.Nil(t, err)

		assert.Equal(t, value, rootBucket.Get(key))

		return nil
	})
	assert.Nil(t, err)
}
