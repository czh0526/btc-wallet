package keystore

import (
	"encoding/binary"
	"fmt"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"golang.org/x/crypto/ripemd160"
	"io"
)

type addrEntry struct {
	pubKeyHash160 [ripemd160.Size]byte
	addr          btcAddress
}

func (e *addrEntry) WriteTo(w io.Writer) (n int64, err error) {
	var written int64

	if written, err = binaryWrite(w, binary.LittleEndian, addrHeader); err != nil {
		return n + written, err
	}
	n += written

	if written, err = binaryWrite(w, binary.LittleEndian, &e.pubKeyHash160); err != nil {
		return n + written, err
	}
	n += written

	written, err = e.addr.WriteTo(w)
	n += written
	return n, err
}

func (e *addrEntry) ReadFrom(r io.Reader) (n int64, err error) {
	var read int64

	if read, err = binaryRead(r, binary.LittleEndian, &e.pubKeyHash160); err != nil {
		return n + read, err
	}

	read, err = e.addr.ReadFrom(r)
	return n + read, err
}

type btcAddress struct {
	store             *Store
	address           btcutil.Address
	flags             addrFlags
	chaincode         [32]byte
	chainIndex        int64
	chainDepth        int64 // unused
	initVector        [16]byte
	privKey           [32]byte
	pubKey            *btcec.PublicKey
	firstSeen         int64
	lastSeen          int64
	firstBlock        int32
	partialSyncHeight int32  // This is reappropriated from armory's `lastBlock` field.
	privKeyCT         []byte // non-nil if unlocked.
}

func (a *btcAddress) WriteTo(w io.Writer) (n int64, err error) {
	return 0, fmt.Errorf("not yet implemented")
}

func (a *btcAddress) ReadFrom(r io.Reader) (n int64, err error) {
	return 0, fmt.Errorf("not yet implemented")
}

type addrFlags struct {
	hasPrivKey              bool
	hasPubkey               bool
	encrypted               bool
	createPrivKeyNextUnlock bool
	compressed              bool
	change                  bool
	unsynced                bool
	partialSync             bool
}
