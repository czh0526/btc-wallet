package main

import (
	"fmt"
	"github.com/czh0526/btc-wallet/chain"
	"github.com/czh0526/btc-wallet/wallet"
	_ "github.com/czh0526/btc-wallet/walletdb/bdb"
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
	fmt.Println("1) 加载配置文件：config => ")
	tcfg, _, err := loadConfig()
	if err != nil {
		return err
	}
	cfg = tcfg

	netDir := networkDir(cfg.AppDataDir, activeNet.Params)
	fmt.Printf("2) 创建数据目录: `%s` \n", netDir)

	fmt.Printf("3) 创建 WalletLoader \n")
	loader := wallet.NewLoader(
		activeNet.Params, netDir, true, cfg.DBTimeout, 250)

	fmt.Println("4) 启动 RPC Server，注册 WalletLoader 服务")
	rpcs, err := startRPCServers(loader)
	if err != nil {
		fmt.Printf("Unable to create RPC servers: %v\n", err)
		return err
	}

	loader.RunAfterLoad(func(w *wallet.Wallet) {
		fmt.Println("5) 向 RPC Server 注册 Wallet 服务")
		startWalletRPCServices(w, rpcs)
	})

	if !cfg.NoInitialLoad {
		fmt.Println("5) 打开数据库")
		_, err = loader.OpenExistingWallet([]byte(cfg.WalletPass), true)
		if err != nil {
			fmt.Printf("open existing wallet failed, err = %v \n", err)
			return err
		}
	}

	addInterruptHandler(func() {
		fmt.Println("6) 卸载数据库")
		err := loader.UnloadWallet()
		if err != nil && err != wallet.ErrNotLoaded {
			fmt.Printf("Failed to close wallet: %v \n", err)
		}
	})

	<-interruptHandlersDone
	fmt.Println("\nShutdown complete")
	return nil
}

func startChainRPC(certs []byte) (*chain.RPCClient, error) {
	fmt.Printf("Attempting RPC client connection to %v", cfg.RPCConnect)
	rpcc, err := chain.NewRPCClient()
	if err != nil {
		return nil, err
	}
	err = rpcc.Start()

}
