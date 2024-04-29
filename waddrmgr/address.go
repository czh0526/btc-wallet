package waddrmgr

import (
	"github.com/btcsuite/btcd/btcutil"
	"github.com/czh0526/btc-wallet/walletdb"
)

type AddressType uint8

const (
	PubKeyHash AddressType = iota
	Script
	RawPubKey
	NestedWitnessPubKey
	WitnessPubKey
	WitnessScript
	TaprootPubKey
	TaprootScript
)

type ManagedAddress interface {
	InternalAccount() uint32

	Address() btcutil.Address

	AddrHash() []byte

	Imported() bool

	Internal() bool

	Compressed() bool

	Used(ns walletdb.ReadBucket) bool

	AddrType() AddressType
}
