package waddrmgr

import (
	"fmt"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/czh0526/btc-wallet/walletdb"
)

const (
	witnessVersionV0 byte = 0x00
	witnessVersionV1 byte = 0x01
)

type witnessScriptAddress struct {
	baseScriptAddress
	address        btcutil.Address
	witnessVersion byte
	isSecretScript bool
}

func (a *witnessScriptAddress) AddrType() AddressType {
	return WitnessScript
}

func (a *witnessScriptAddress) Address() btcutil.Address {
	return a.address
}

func (a *witnessScriptAddress) AddrHash() []byte {
	return a.address.ScriptAddress()
}

func (a *witnessScriptAddress) Compressed() bool {
	return true
}

func (a *witnessScriptAddress) Used(ns walletdb.ReadBucket) bool {
	return a.manager.fetchUsed(ns, a.AddrHash())
}

func (a *witnessScriptAddress) Script() ([]byte, error) {
	if a.isSecretScript && a.manager.rootManager.WatchOnly() {
		return nil, managerError(ErrWatchingOnly, errWatchingOnly, nil)
	}

	a.manager.mtx.Lock()
	defer a.manager.mtx.Unlock()

	if a.isSecretScript && a.manager.rootManager.IsLocked() {
		return nil, managerError(ErrLocked, errLocked, nil)
	}

	cryptoKey := a.manager.rootManager.cryptoKeyScript
	if !a.isSecretScript {
		cryptoKey = a.manager.rootManager.cryptoKeyPub
	}

	return a.unlock(cryptoKey)
}

var _ ManagedScriptAddress = (*witnessScriptAddress)(nil)

type taprootScriptAddress struct {
	witnessScriptAddress
	TweakedPubKey *btcec.PublicKey
}

func (a *taprootScriptAddress) AddrType() AddressType {
	return TaprootScript
}

func (a *taprootScriptAddress) Address() btcutil.Address {
	return a.address
}

func (a *taprootScriptAddress) AddrHash() []byte {
	return schnorr.SerializePubKey(a.TweakedPubKey)
}

func (a *taprootScriptAddress) TaprootScript() (*Tapscript, error) {
	script, err := a.Script()
	if err != nil {
		return nil, err
	}

	return tlvDecodeTaprootTaprootScript(script)
}

var _ ManagedTaprootScriptAddress = (*taprootScriptAddress)(nil)

func newWitnessScriptAddress(m *ScopedKeyManager, account uint32,
	scriptIdent, scriptEncrypted []byte,
	witnessVersion byte, isSecretScript bool) (ManagedScriptAddress, error) {

	switch witnessVersion {
	case witnessVersionV0:
		address, err := btcutil.NewAddressWitnessScriptHash(
			scriptIdent, m.rootManager.chainParams)
		if err != nil {
			return nil, err
		}

		return &witnessScriptAddress{
			baseScriptAddress: baseScriptAddress{
				manager:         m,
				account:         account,
				scriptEncrypted: scriptEncrypted,
			},
			address:        address,
			witnessVersion: witnessVersion,
			isSecretScript: isSecretScript,
		}, nil

	case witnessVersionV1:
		address, err := btcutil.NewAddressTaproot(
			scriptIdent, m.rootManager.chainParams)
		if err != nil {
			return nil, err
		}

		tweakedPubKey, err := schnorr.ParsePubKey(scriptIdent)
		if err != nil {
			return nil, fmt.Errorf("error lifting public key from script ident: %w", err)
		}

		return &taprootScriptAddress{
			witnessScriptAddress: witnessScriptAddress{
				baseScriptAddress: baseScriptAddress{
					manager:         m,
					account:         account,
					scriptEncrypted: scriptEncrypted,
				},
				address:        address,
				witnessVersion: witnessVersion,
				isSecretScript: isSecretScript,
			},
			TweakedPubKey: tweakedPubKey,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported witness version: %d", witnessVersion)
	}
}
