package wallet

import (
	"fmt"
	"os"
)

func CreateDir(parent string) (string, error) {
	walletDir := fmt.Sprintf("%s/btc-wallet", parent)
	_, err := os.Stat(walletDir)
	if err == nil {
		// 已存在
		return walletDir, nil
	}

	if os.IsNotExist(err) {
		// 不存在，创建
		err = os.Mkdir(walletDir, os.ModePerm)
		if err == nil {
			return walletDir, nil
		}
	}

	return "", fmt.Errorf("创建钱包所在文件夹失败：err = %v", err)
}
