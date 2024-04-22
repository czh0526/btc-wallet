package main

import (
	"fmt"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/czh0526/btc-wallet/key"
	"github.com/czh0526/btc-wallet/seed"
	"os"

	"github.com/czh0526/btc-wallet/wallet"
)

func main() {
	currDir, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("获取当前路径失败：err = %v", err))
	}

	walletDir, err := wallet.CreateDir(currDir)
	if err != nil {
		panic(fmt.Sprintf("创建钱包文件夹失败：err = %v", err))
	}

	walletDB, isNew, err := wallet.OpenDB(walletDir)
	if err != nil {
		panic(fmt.Sprintf("创建钱包数据库失败：err = %v", err))
	}

	if isNew {
		var seedBytes []byte
		var rootKey *hdkeychain.ExtendedKey

		seedBytes, err = seed.NewSeed()
		if err != nil {
			panic(fmt.Sprintf("生成随机数失败: %v", err))
		}

		rootKey, err = key.NewRootKey(seedBytes)
		if err != nil {
			panic(fmt.Sprintf("生成`RootKey`失败: %v", err))
		}
	}

	defer func() {
		err = walletDB.Close()
		if err != nil {
			panic(fmt.Sprintf("关闭钱包数据库失败：err = %v", err))
		}
	}()
}
