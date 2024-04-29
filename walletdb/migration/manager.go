package migration

import "github.com/czh0526/btc-wallet/walletdb"

type Version struct {
	Number    uint32
	Migration func(bucket walletdb.ReadWriteBucket) error
}
