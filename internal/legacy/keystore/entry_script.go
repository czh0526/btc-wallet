package keystore

import (
	"fmt"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/txscript"
	"golang.org/x/crypto/ripemd160"
	"io"
)

type scriptEntry struct {
	scriptHash160 [ripemd160.Size]byte
	script        scriptAddress
}

type scriptAddress struct {
	store             *Store
	address           btcutil.Address
	class             txscript.ScriptClass
	addresses         []btcutil.Address
	reqSigs           int
	flags             scriptFlags
	script            p2SHScript // variable length
	firstSeen         int64
	lastSeen          int64
	firstBlock        int32
	partialSyncHeight int32
}

func (e *scriptEntry) WriteTo(w io.Writer) (n int64, err error) {
	return 0, fmt.Errorf("not yet implemented")
}

func (e *scriptEntry) ReadFrom(r io.Reader) (n int64, err error) {
	return 0, fmt.Errorf("not yet implemented")
}

type scriptFlags struct {
	hasScript   bool
	change      bool
	unsynced    bool
	partialSync bool
}

type p2SHScript []byte
