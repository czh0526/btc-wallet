package key

import (
	"fmt"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/czh0526/btc-wallet/netparams"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewMaster(t *testing.T) {
	hdSeed, err := hdkeychain.GenerateSeed(hdkeychain.RecommendedSeedLen)
	assert.NoError(t, err)

	netParams := netparams.MainNetParams
	params := netParams.Params

	masterKey, err := hdkeychain.NewMaster(hdSeed, params)
	assert.NoError(t, err)
	fmt.Printf("master key \t=> %v \n", masterKey)

	addr, err := masterKey.Address(params)
	assert.NoError(t, err)
	fmt.Printf("\t address => %v \n", addr)

	neuter, err := masterKey.Neuter()
	assert.NoError(t, err)
	fmt.Printf("\t neuter => %v \n", neuter)
}
