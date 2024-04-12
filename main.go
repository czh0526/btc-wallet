package main

import (
	"os"
)

func main() {
	os.Mkdir("btc-wallet", os.ModePerm)

	walletSetup := &walletsetup.Create{}

}
