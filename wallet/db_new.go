package main

import (
	"fmt"
	"github.com/btcsuite/btcwallet/walletdb"
	_ "github.com/btcsuite/btcwallet/walletdb/bdb"
	"os"
)

func NewDB(dir string) (walletdb.DB, error) {

	walletDir := fmt.Sprintf("%s/btc-wallet", dir)
	err := os.Mkdir(walletDir, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("创建钱包所在文件夹失败：err = %v", err)
	}

	walletPath := fmt.Sprintf("%s/wallet.db", walletDir)
	db, err := walletdb.Create("bdb", walletPath)
	if err != nil {
		return nil, fmt.Errorf("创建钱包数据库失败：err = %v", err)
	}

	return db, nil
}
