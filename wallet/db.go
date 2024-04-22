package wallet

import (
	"fmt"
	"github.com/btcsuite/btcwallet/walletdb"
	_ "github.com/btcsuite/btcwallet/walletdb/bdb"
	"os"
	"time"
)

func OpenDB(dir string) (walletdb.DB, bool, error) {

	dbPath := fmt.Sprintf("%s/wallet.db", dir)
	_, err := os.Stat(dbPath)
	var db walletdb.DB

	// 新建
	if os.IsNotExist(err) {
		db, err = walletdb.Create("bdb", dbPath, true, 60*time.Second)
		if err != nil {
			return nil, true, fmt.Errorf("新建钱包数据库失败：err = %v", err)
		}

		return db, true, nil
	}

	// 打开
	db, err = walletdb.Open("bdb", dbPath, true, 60*time.Second)
	if err != nil {
		return nil, false, fmt.Errorf("打开钱包数据库失败：err = %v", err)
	}
	return db, false, nil
}
