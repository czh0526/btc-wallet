package waddrmgr

import (
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/txscript"
)

type TapscriptType uint8

const (
	TapscriptTypeFullTree      TapscriptType = 0
	TapscriptTypePartialReveal TapscriptType = 1
	TaprootKeySpendRootHash    TapscriptType = 2
	TaprootFullKeyOnly         TapscriptType = 3
)

type Tapscript struct {
	Type           TapscriptType
	ControlBlock   *txscript.ControlBlock
	Leaves         []txscript.TapLeaf
	RevealedScript []byte
	RootHash       []byte
	FullOutputKey  *btcec.PublicKey
}
