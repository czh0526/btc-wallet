package waddrmgr

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
