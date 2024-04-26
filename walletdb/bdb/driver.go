package bdb

import (
	"fmt"
	"github.com/czh0526/btc-wallet/walletdb"
	"time"
)

const (
	dbType = "bdb"
)

func parseArgs(funcName string,
	args ...interface{}) (string, bool, time.Duration, error) {

	if len(args) != 3 {
		return "", false, 0, fmt.Errorf("invalid arguments to %s.%s "+
			"-- expected database path, no-freelist-sync and "+
			"timeout option",
			dbType, funcName)
	}

	dbPath, ok := args[0].(string)
	if !ok {
		return "", false, 0, fmt.Errorf("first argument to %s.%s is "+
			"invalid -- expected database path string", dbType,
			funcName)
	}

	noFreelistSync, ok := args[1].(bool)
	if !ok {
		return "", false, 0, fmt.Errorf("second argument to %s.%s is "+
			"invalid -- expected no-freelist-sync bool", dbType,
			funcName)
	}

	timeout, ok := args[2].(time.Duration)
	if !ok {
		return "", false, 0, fmt.Errorf("third argument to %s.%s is "+
			"invalid -- expected timeout time.Duration", dbType,
			funcName)
	}

	return dbPath, noFreelistSync, timeout, nil
}

func openDBDriver(args ...interface{}) (walletdb.DB, error) {
	dbPath, noFreelistSync, timeout, err := parseArgs("Open", args...)
	if err != nil {
		return nil, err
	}

	return openDB(dbPath, noFreelistSync, false, timeout)
}

func createDBDriver(args ...interface{}) (walletdb.DB, error) {
	dbPath, noFreelistSync, timeout, err := parseArgs("Create", args...)
	if err != nil {
		return nil, err
	}

	return openDB(dbPath, noFreelistSync, true, timeout)
}

func init() {
	driver := walletdb.Driver{
		DBType: dbType,
		Create: createDBDriver,
		Open:   openDBDriver,
	}

	if err := walletdb.RegisterDriver(driver); err != nil {
		panic(fmt.Sprintf("Failed to register database driver '%s': %v", dbType, err))
	}
}
