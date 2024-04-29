package waddrmgr

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"time"
)

type BlockStamp struct {
	Height    int32
	Hash      chainhash.Hash
	Timestamp time.Time
}

type syncState struct {
	startBlock BlockStamp
	syncedTo   BlockStamp
}

func newSyncState(startBlock *BlockStamp, syncedTo *BlockStamp) *syncState {
	return &syncState{
		startBlock: *startBlock,
		syncedTo:   *syncedTo,
	}
}
