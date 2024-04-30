package main

import (
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/czh0526/btc-wallet/internal/legacy/keystore"
	"github.com/czh0526/btc-wallet/internal/prompt"
	"github.com/czh0526/btc-wallet/wallet"
	"os"
	"path/filepath"
	"time"
)

func networkDir(dataDir string, chainParams *chaincfg.Params) string {
	netname := chainParams.Name

	if chainParams.Net == wire.TestNet3 {
		netname = "testnet"
	}

	return filepath.Join(dataDir, netname)
}

func createWallet(cfg *config) error {

	netDir := networkDir(cfg.AppDataDir, activeNet.Params)
	loader := wallet.NewLoader(
		activeNet.Params, netDir, true, cfg.DBTimeout, 250)

	keystorePath := filepath.Join(netDir, "wallet.bin")
	var legacyKeyStore *keystore.Store
	_, err := os.Stat(keystorePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	} else if err == nil {
		legacyKeyStore, err = keystore.OpenDir(netDir)
		if err != nil {
			return err
		}
	}

	if legacyKeyStore != nil {

	}

	privPass, _ := prompt.PrivatePass()
	pubPass, _ := prompt.PublicPass()
	seed, _ := prompt.Seed()

	fmt.Println("Creating the wallet...")
	w, err := loader.CreateNewWallet(pubPass, privPass, seed, time.Now())
	if err != nil {
		return err
	}

	w.Manager.Close()
	fmt.Println("The wallet has been created successfully!")
	return nil
}
