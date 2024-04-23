package netparams

import "github.com/btcsuite/btcd/chaincfg"

type Params struct {
	*chaincfg.Params
	RPCClientPort string
	RPCServerPort string
}

var MainNetParams = Params{
	Params:        &chaincfg.MainNetParams,
	RPCClientPort: "8334",
	RPCServerPort: "8332",
}

var TestNetParams = Params{
	Params:        &chaincfg.TestNet3Params,
	RPCClientPort: "18334",
	RPCServerPort: "18332",
}
