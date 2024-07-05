package waddrmgr

import (
	"fmt"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/czh0526/btc-wallet/internal/zero"
	"github.com/czh0526/btc-wallet/walletdb"
	"sync"
)

type baseScriptAddress struct {
	manager         *ScopedKeyManager
	account         uint32
	address         *btcutil.AddressScriptHash
	scriptEncrypted []byte
	scriptClearText []byte
	scriptMutex     sync.Mutex
}

func (a *baseScriptAddress) InternalAccount() uint32 {
	return a.account
}

func (a *baseScriptAddress) Imported() bool {
	return true
}

func (a *baseScriptAddress) Internal() bool {
	return false
}

func (a *baseScriptAddress) lock() {
	a.scriptMutex.Lock()
	defer a.scriptMutex.Unlock()

	zero.Bytes(a.scriptClearText)
	a.scriptClearText = nil
}

func (a *baseScriptAddress) unlock(key EncryptorDecryptor) ([]byte, error) {
	a.scriptMutex.Lock()
	defer a.scriptMutex.Unlock()

	if len(a.scriptClearText) == 0 {
		script, err := key.Decrypt(a.scriptClearText)
		if err != nil {
			str := fmt.Sprintf("failed to decrypt script for %s", a.address)
			return nil, managerError(ErrCrypto, str, err)
		}

		a.scriptClearText = script
	}

	scriptCopy := make([]byte, len(a.scriptClearText))
	copy(scriptCopy, a.scriptClearText)
	return scriptCopy, nil
}

type scriptAddress struct {
	baseScriptAddress
	address *btcutil.AddressScriptHash
}

func (a *scriptAddress) AddrType() AddressType {
	return Script
}

func (a *scriptAddress) Address() btcutil.Address {
	return a.address
}

func (a *scriptAddress) AddrHash() []byte {
	return a.address.Hash160()[:]
}

func (a *scriptAddress) Compressed() bool {
	return false
}

func (a *scriptAddress) Used(ns walletdb.ReadBucket) bool {
	return a.manager.fetchUsed(ns, a.AddrHash())
}

func (a *scriptAddress) Script() ([]byte, error) {
	if a.manager.rootManager.WatchOnly() {
		return nil, managerError(ErrWatchingOnly, errWatchingOnly, nil)
	}

	a.manager.mtx.Lock()
	defer a.manager.mtx.Unlock()

	if a.manager.rootManager.IsLocked() {
		return nil, managerError(ErrLocked, errLocked, nil)
	}

	return a.unlock(a.manager.rootManager.cryptoKeyScript)
}

func newScriptAddress(m *ScopedKeyManager, account uint32,
	scriptHash, scriptEncrypted []byte) (*scriptAddress, error) {

	address, err := btcutil.NewAddressScriptHashFromHash(
		scriptHash, m.rootManager.chainParams)
	if err != nil {
		return nil, err
	}

	return &scriptAddress{
		baseScriptAddress: baseScriptAddress{
			manager:         m,
			account:         account,
			scriptEncrypted: scriptEncrypted,
		},
		address: address,
	}, nil
}
