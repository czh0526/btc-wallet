package main

import (
	"fmt"
	"github.com/czh0526/btc-wallet/internal/cfgutil"
	"github.com/czh0526/btc-wallet/wallet"
	"os"
	"path/filepath"
	"time"
)

type config struct {
	ConfigFile      string        `short:"C" long:"configfile" description:"Path to configuration file"`
	ShowVersion     bool          `short:"V" long:"version" description:"Display version information and exit"`
	Create          bool          `long:"create" description:"Create the wallet if it does not exist"`
	CreateTemp      bool          `long:"createtemp" description:"Create a temporary simulation wallet (pass=password) in the data directory indicated; must call with --datadir"`
	AppDataDir      string        `short:"A" long:"appdata" description:"Application data directory for wallet config, databases and logs"`
	TestNet3        bool          `long:"testnet" description:"Use the test Bitcoin network (version 3) (default mainnet)"`
	SimNet          bool          `long:"simnet" description:"Use the simulation test network (default mainnet)"`
	SigNet          bool          `long:"signet" description:"Use the signet test network (default mainnet)"`
	SigNetChallenge string        `long:"signetchallenge" description:"Connect to a custom signet network defined by this challenge instead of using the global default signet test network -- Can be specified multiple times"`
	SigNetSeedNode  []string      `long:"signetseednode" description:"Specify a seed node for the signet network instead of using the global default signet network seed nodes"`
	NoInitialLoad   bool          `long:"noinitialload" description:"Defer wallet creation/opening on startup and enable loading wallets over RPC"`
	DebugLevel      string        `short:"d" long:"debuglevel" description:"Logging level {trace, debug, info, warn, error, critical}"`
	LogDir          string        `long:"logdir" description:"Directory to log output."`
	Profile         string        `long:"profile" description:"Enable HTTP profiling on given port -- NOTE port must be between 1024 and 65536"`
	DBTimeout       time.Duration `long:"dbtimeout" description:"The timeout value to use when opening the wallet database."`

	// Wallet options
	WalletPass string `long:"walletpass" default-mask:"-" description:"The public wallet password -- Only required if the wallet was created with one"`

	// RPC client options
	RPCConnect       string `short:"c" long:"rpcconnect" description:"Hostname/IP and port of btcd RPC server to connect to (default localhost:8334, testnet: localhost:18334, simnet: localhost:18556)"`
	CAFile           string `long:"cafile" description:"File containing root certificates to authenticate a TLS connections with btcd"`
	DisableClientTLS bool   `long:"noclienttls" description:"Disable TLS for the RPC client -- NOTE: This is only allowed if the RPC client is connecting to localhost"`
	BtcdUsername     string `long:"btcdusername" description:"Username for btcd authentication"`
	BtcdPassword     string `long:"btcdpassword" default-mask:"-" description:"Password for btcd authentication"`
	Proxy            string `long:"proxy" description:"Connect via SOCKS5 proxy (eg. 127.0.0.1:9050)"`
	ProxyUser        string `long:"proxyuser" description:"Username for proxy server"`
	ProxyPass        string `long:"proxypass" default-mask:"-" description:"Password for proxy server"`

	// SPV client options
	UseSPV       bool          `long:"usespv" description:"Enables the experimental use of SPV rather than RPC for chain synchronization"`
	AddPeers     []string      `short:"a" long:"addpeer" description:"Add a peer to connect with at startup"`
	ConnectPeers []string      `long:"connect" description:"Connect only to the specified peers at startup"`
	MaxPeers     int           `long:"maxpeers" description:"Max number of inbound and outbound peers"`
	BanDuration  time.Duration `long:"banduration" description:"How long to ban misbehaving peers.  Valid time units are {s, m, h}.  Minimum 1 second"`
	BanThreshold uint32        `long:"banthreshold" description:"Maximum allowed ban score before disconnecting and banning misbehaving peers."`

	// RPC server options
	//
	// The legacy server is still enabled by default (and eventually will be
	// replaced with the experimental server) so prepare for that change by
	// renaming the struct fields (but not the configuration options).
	//
	// Usernames can also be used for the consensus RPC client, so they
	// aren't considered legacy.
	RPCCert                string   `long:"rpccert" description:"File containing the certificate file"`
	RPCKey                 string   `long:"rpckey" description:"File containing the certificate key"`
	OneTimeTLSKey          bool     `long:"onetimetlskey" description:"Generate a new TLS certpair at startup, but only write the certificate to disk"`
	DisableServerTLS       bool     `long:"noservertls" description:"Disable TLS for the RPC server -- NOTE: This is only allowed if the RPC server is bound to localhost"`
	LegacyRPCListeners     []string `long:"rpclisten" description:"Listen for legacy RPC connections on this interface/port (default port: 8332, testnet: 18332, simnet: 18554)"`
	LegacyRPCMaxClients    int64    `long:"rpcmaxclients" description:"Max number of legacy RPC clients for standard connections"`
	LegacyRPCMaxWebsockets int64    `long:"rpcmaxwebsockets" description:"Max number of legacy RPC websocket connections"`
	Username               string   `short:"u" long:"username" description:"Username for legacy RPC and btcd authentication (if btcdusername is unset)"`
	Password               string   `short:"P" long:"password" default-mask:"-" description:"Password for legacy RPC and btcd authentication (if btcdpassword is unset)"`

	// EXPERIMENTAL RPC server options
	//
	// These options will change (and require changes to config files, etc.)
	// when the new gRPC server is enabled.
	ExperimentalRPCListeners []string `long:"experimentalrpclisten" description:"Listen for RPC connections on this interface/port"`

	// Deprecated options
	DataDir string `short:"b" long:"datadir" default-mask:"-" description:"DEPRECATED -- use appdata instead"`
}

func loadConfig() (*config, []string, error) {
	cfg := config{
		DebugLevel:             "info",
		ConfigFile:             "/Users/zhihongcai/Library/Application Support/Btcwallet/btcwallet.conf",
		AppDataDir:             "/Users/zhihongcai/Library/Application Support/Btcwallet",
		LogDir:                 "/Users/zhihongcai/Library/Application Support/Btcwallet/logs/mainnet",
		WalletPass:             "public",
		CAFile:                 "/Users/zhihongcai/Library/Application Support/Btcd/rpc.cert",
		RPCKey:                 "/Users/zhihongcai/Library/Application Support/Btcwallet/rpc.key",
		RPCCert:                "/Users/zhihongcai/Library/Application Support/Btcwallet/rpc.cert",
		LegacyRPCMaxClients:    10,
		LegacyRPCMaxWebsockets: 25,
		DataDir:                "/Users/zhihongcai/Library/Application Support/Btcwallet",
		UseSPV:                 false,
		AddPeers:               []string{},
		ConnectPeers:           []string{},
		MaxPeers:               125,
		BanDuration:            time.Hour * 24,
		BanThreshold:           uint32(100),
		DBTimeout:              60 * time.Second,
	}

	netDir := networkDir(cfg.AppDataDir, activeNet.Params)
	dbPath := filepath.Join(netDir, "wallet.db")

	dbFileExists, err := cfgutil.FileExists(dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, nil, err
	}

	if cfg.Create {
		if dbFileExists {
			err := fmt.Errorf("the wallet database file `%v` already exists", dbPath)
			fmt.Fprintln(os.Stderr, err)
			return nil, nil, err
		}

		if err := wallet.CheckCreateDir(netDir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return nil, nil, err
		}

		if err := createWallet(&cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Unable to create wallet: %v \n", err)
			return nil, nil, err
		}

		os.Exit(0)

	} else if !dbFileExists && !cfg.NoInitialLoad {
		keystorePath := filepath.Join(netDir, "wallet.bin")
		keystoreExists, err := cfgutil.FileExists(keystorePath)
		if err != nil {
			fmt.Println(os.Stderr, err)
			return nil, nil, err
		}
		if !keystoreExists {
			err = fmt.Errorf("the wallet does not exist, run with --create option to initialize and create it")
		} else {
			err = fmt.Errorf("the wallet is in legacy format, run with --create option to import it")
		}
		fmt.Fprintln(os.Stderr, err)
		return nil, nil, err
	}

	return &cfg, nil, nil
}
