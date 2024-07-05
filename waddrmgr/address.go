package waddrmgr

import (
	"fmt"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/czh0526/btc-wallet/walletdb"
)

var (
	ErrPubKeyMismatch = fmt.Errorf("derived pubkey doesn't match original")

	ErrAddrMismatch = fmt.Errorf("derived addr doesn't match original")

	ErrInvalidSignature = fmt.Errorf("private key sig doesn't validate against pubkey")
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

type ValidatableManagedAddress interface {
	ManagedPubKeyAddress

	Validate(msg [32]byte, priv *btcec.PrivateKey) error
}

type ManagedPubKeyAddress interface {
	ManagedAddress

	PubKey() *btcec.PublicKey

	ExportPubKey() string

	PrivKey() (*btcec.PrivateKey, error)

	ExportPrivKey() (*btcutil.WIF, error)

	DerivationInfo() (KeyScope, DerivationPath, bool)
}

type ManagedScriptAddress interface {
	ManagedAddress

	Script() ([]byte, error)
}

type ManagedTaprootScriptAddress interface {
	ManagedScriptAddress
	TaprootScript() (*Tapscript, error)
}

var _ ManagedPubKeyAddress = (*managedAddress)(nil)

var _ ManagedScriptAddress = (*scriptAddress)(nil)

type signature interface {
	Verify([]byte, *btcec.PublicKey) bool
}
