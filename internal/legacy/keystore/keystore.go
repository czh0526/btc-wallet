package keystore

import (
	"fmt"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const (
	Filename = "wallet.bin"
)

type version struct {
	major         byte
	minor         byte
	bugfix        byte
	autoincrement byte
}

type netParams chaincfg.Params

type walletFlags struct {
	useEncryption bool
	watchingOnly  bool
}

type addressKey string

type SyncStatus interface {
	ImplementsSyncStatus()
}

type WalletAddress interface {
	Address() btcutil.Address
	AddrHash() string
	FirstBlock() int32
	Imported() bool
	Change() bool
	Compressed() bool
	SyncStatus() SyncStatus
}

type walletAddress interface {
	io.ReaderFrom
	io.WriterTo
	WalletAddress
}

type Store struct {
	dirty bool
	path  string
	dir   string
	file  string

	mtx   sync.RWMutex
	vers  version
	net   *netParams
	flags walletFlags

	addrMap     map[addressKey]walletAddress
	chainIdxMap map[int64]btcutil.Address
}

func OpenDir(dir string) (*Store, error) {
	path := filepath.Join(dir, Filename)
	fi, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer fi.Close()

	store := new(Store)
	_, err = store.ReadFrom(fi)
	if err != nil {
		return nil, err
	}

	store.path = path
	store.dir = dir
	store.file = Filename
	return store, nil
}

func (s *Store) ReadFrom(r io.Reader) (n int64, err error) {
	return 0, fmt.Errorf("not yet implemented")
}

func (s *Store) WriteTo(w io.Writer) (n int64, err error) {
	return 0, fmt.Errorf("not yet implemented")
}
