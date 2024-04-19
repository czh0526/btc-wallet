package main

import (
	"fmt"
	"os"
)

func main() {
	currDir, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("获取当前路径失败：err = %v"))
	}

	walletdb := wallet.NewDB(currDir)
}
