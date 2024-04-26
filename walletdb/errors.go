package walletdb

import (
	"errors"
)

var (
	ErrDbTypeRegistered = errors.New("database type already registered")

	ErrDbUnknownType = errors.New("unknown database type")

	ErrDbDoesNotExist = errors.New("database does not exist")

	ErrDbExists = errors.New("database already exists")

	ErrDbNotOpen = errors.New("database not open")

	ErrDbAlreadyOpen = errors.New("database already open")

	ErrInvalid = errors.New("invalid database")

	ErrDryRunRollBack = errors.New("dry run only; should roll back")
)

var (
	// ErrTxClosed is returned when attempting to commit or rollback a
	// transaction that has already had one of those operations performed.
	ErrTxClosed = errors.New("tx closed")

	// ErrTxNotWritable is returned when an operation that requires write
	// access to the database is attempted against a read-only transaction.
	ErrTxNotWritable = errors.New("tx not writable")
)

// Errors that can occur when putting or deleting a value or bucket.
var (
	// ErrBucketNotFound is returned when trying to access a bucket that has
	// not been created yet.
	ErrBucketNotFound = errors.New("bucket not found")

	// ErrBucketExists is returned when creating a bucket that already exists.
	ErrBucketExists = errors.New("bucket already exists")

	// ErrBucketNameRequired is returned when creating a bucket with a blank name.
	ErrBucketNameRequired = errors.New("bucket name required")

	// ErrKeyRequired is returned when inserting a zero-length key.
	ErrKeyRequired = errors.New("key required")

	// ErrKeyTooLarge is returned when inserting a key that is larger than MaxKeySize.
	ErrKeyTooLarge = errors.New("key too large")

	// ErrValueTooLarge is returned when inserting a value that is larger than MaxValueSize.
	ErrValueTooLarge = errors.New("value too large")

	// ErrIncompatibleValue is returned when trying create or delete a
	// bucket on an existing non-bucket key or when trying to create or
	// delete a non-bucket key on an existing bucket key.
	ErrIncompatibleValue = errors.New("incompatible value")
)
