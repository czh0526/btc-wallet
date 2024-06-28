package chain

import (
	"errors"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
)

type RPCClient struct {
	*rpcclient.Client
	connConfig  *rpcclient.ConnConfig
	chainParams *chaincfg.Params
}

func NewRPCClient(chainParams *chaincfg.Params, connect, user, pass string, certs []byte,
	disableTLS bool, reconnectAttempts int) (*RPCClient, error) {

	if reconnectAttempts <= 0 {
		return nil, errors.New("reconnectAttempts must be a positive")
	}

	client := &RPCClient{
		connConfig: &rpcclient.ConnConfig{
			Host:                 connect,
			Endpoint:             "ws",
			User:                 user,
			Pass:                 pass,
			Certificates:         certs,
			DisableAutoReconnect: false,
			DisableConnectOnNew:  true,
			DisableTLS:           disableTLS,
		},
		chainParams: chainParams,
	}
	ntfnCallbacks := &rpcclient.NotificationHandlers{
		OnClientConnected:   nil,
		OnBlockConnected:    nil,
		OnBlockDisconnected: nil,
		OnRecvTx:            nil,
		OnRedeemingTx:       nil,
		OnRescanFinished:    nil,
		OnRescanProgress:    nil,
	}
	rpcClient, err := rpcclient.New(client.connConfig, ntfnCallbacks)
	if err != nil {
		return nil, err
	}
	client.Client = rpcClient
	return client, nil
}
