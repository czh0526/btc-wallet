package main

import (
	"fmt"
	"github.com/czh0526/btc-wallet/wallet"
	"os"
	"runtime"
)

var (
	cfg *config
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	if err := walletMain(); err != nil {
		os.Exit(1)
	}
}

func walletMain() error {
	tcfg, _, err := loadConfig()
	if err != nil {
		return err
	}
	cfg = tcfg

	netDir := networkDir(cfg.AppDataDir, activeNet.Params)
	loader := wallet.NewLoader(
		activeNet.Params, netDir, true, cfg.DBTimeout, 250)

	if !cfg.NoInitialLoad {
		_, err = loader.OpenExistingWallet([]byte(cfg.WalletPass), true)
		if err != nil {
			fmt.Printf("open existing wallet failed, err = %v \n", err)
			return err
		}
	}

	addInterruptHandler(func() {
		err := loader.UnloadWallet()
		if err != nil && err != wallet.ErrNotLoaded {
			fmt.Printf("Failed to close wallet: %v \n", err)
		}
	})

	<-interruptHandlersDone
	fmt.Println("Shutdown complete")
	return nil
}
