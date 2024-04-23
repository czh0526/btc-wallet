package keystore

import (
	"os"
	"path/filepath"
)

const (
	Filename = "wallet.bin"
)

type Store struct {
	dirty bool
	path  string
	dir   string
	file  string
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

}
