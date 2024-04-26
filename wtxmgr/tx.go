package wtxmgr

import (
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/czh0526/btc-wallet/walletdb"
)

type Store struct {
	chainParams *chaincfg.Params
}

func Create(ns walletdb.ReadWriteBucket) error {
	fmt.Println("wtxmgr.Create() has not been implemented yet")
	return nil
}

func Open(ns walletdb.ReadWriteBucket, chainParams *chaincfg.Params) (*Store, error) {
	fmt.Println("wtxmgr.Open() has not been implemented yet")
	return &Store{}, nil
}
