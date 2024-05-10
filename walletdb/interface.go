package walletdb

import (
	"io"
)

type Driver struct {
	DBType string
	Create func(args ...interface{}) (DB, error)
	Open   func(args ...interface{}) (DB, error)
}

type DB interface {
	BeginReadTx() (ReadTx, error)
	BeginReadWriteTx() (ReadWriteTx, error)
	Copy(w io.Writer) error
	Close() error
	PrintStats() string
	View(f func(tx ReadTx) error, reset func()) error
	Update(f func(tx ReadWriteTx) error, reset func()) error
}

type ReadTx interface {
	ReadBucket(key []byte) ReadBucket
	ForEachBucket(func(key []byte) error) error
	Rollback() error
}

type ReadWriteTx interface {
	ReadTx

	ReadWriteBucket(key []byte) ReadWriteBucket
	CreateTopLevelBucket(key []byte) (ReadWriteBucket, error)
	DeleteTopLevelBucket(key []byte) error

	Commit() error
	OnCommit(func())
}

type ReadBucket interface {
	Name() []byte
	NestedReadBucket(key []byte) ReadBucket
	ForEach(func(k, v []byte) error) error
	Get(key []byte) []byte
	ReadCursor() ReadCursor
}

type ReadWriteBucket interface {
	ReadBucket

	NestedReadWriteBucket(key []byte) ReadWriteBucket
	CreateBucket(key []byte) (ReadWriteBucket, error)
	CreateBucketIfNotExists(key []byte) (ReadWriteBucket, error)
	DeleteNestedBucket(key []byte) error
	Put(key, value []byte) error
	Delete(key []byte) error
	ReadWriteCursor() ReadWriteCursor
	Tx() ReadWriteTx
	NextSequence() (uint64, error)
	SetSequence(v uint64) error
	Sequence() uint64
}

type ReadCursor interface {
	First() (key, value []byte)
	Last() (key, value []byte)
	Next() (key, value []byte)
	Prev() (key, value []byte)
	Seek(seek []byte) (key, value []byte)
}

type ReadWriteCursor interface {
	ReadCursor

	Delete() error
}

func Create(dbType string, args ...interface{}) (DB, error) {
	drv, exists := drivers[dbType]
	if !exists {
		return nil, ErrDbUnknownType
	}

	return drv.Create(args...)
}

func Open(dbType string, args ...interface{}) (DB, error) {
	drv, exists := drivers[dbType]
	if !exists {
		return nil, ErrDbUnknownType
	}

	return drv.Open(args...)
}

func View(db DB, f func(tx ReadTx) error) error {
	return db.View(f, func() {})
}

func Update(db DB, f func(tx ReadWriteTx) error) error {
	return db.Update(f, func() {})
}

var drivers = make(map[string]*Driver)

func RegisterDriver(driver Driver) error {
	if _, exists := drivers[driver.DBType]; exists {
		return ErrDbTypeRegistered
	}

	drivers[driver.DBType] = &driver
	return nil
}
