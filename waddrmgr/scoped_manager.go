package waddrmgr

import (
	"fmt"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/lightninglabs/neutrino/cache/lru"
	"sync"
)

const (
	defaultPrivKeyCacheSize = 10_000
)

type KeyScope struct {
	Purpose uint32
	Coin    uint32
}

type ScopeAddrSchema struct {
	ExternalAddrType AddressType
	InternalAddrType AddressType
}

var (
	KeyScopeBIP0049Plus = KeyScope{
		Purpose: 49,
		Coin:    0,
	}

	KeyScopeBIP0084 = KeyScope{
		Purpose: 84,
		Coin:    0,
	}

	KeyScopeBIP0086 = KeyScope{
		Purpose: 86,
		Coin:    0,
	}

	KeyScopeBIP0044 = KeyScope{
		Purpose: 44,
		Coin:    0,
	}

	DefaultKeyScopes = []KeyScope{
		KeyScopeBIP0049Plus,
		KeyScopeBIP0084,
		KeyScopeBIP0086,
		KeyScopeBIP0044,
	}

	ScopeAddrMap = map[KeyScope]ScopeAddrSchema{
		KeyScopeBIP0049Plus: {
			ExternalAddrType: NestedWitnessPubKey,
			InternalAddrType: WitnessPubKey,
		},
		KeyScopeBIP0084: {
			ExternalAddrType: WitnessPubKey,
			InternalAddrType: WitnessPubKey,
		},
		KeyScopeBIP0086: {
			ExternalAddrType: TaprootPubKey,
			InternalAddrType: TaprootPubKey,
		},
		KeyScopeBIP0044: {
			ExternalAddrType: PubKeyHash,
			InternalAddrType: PubKeyHash,
		},
	}
)

type accountInfo struct {
	acctName string
	acctType accountType

	acctKeyEncrypted []byte
	acctKeyPriv      *hdkeychain.ExtendedKey
	acctKeyPub       *hdkeychain.ExtendedKey

	nextExternalIndex uint32
	lastExternalAddr  ManagedAddress

	nextInternalAddr uint32
	lastInternalasr  ManagedAddress

	addrSchema           *ScopeAddrSchema
	masterKeyFingerprint uint32
}

type unlockDeriveInfo struct {
	managedAddr ManagedAddress
	branch      uint32
	index       uint32
}

type DerivationPath struct {
	InternalAccount      uint32
	Account              uint32
	Branch               uint32
	Index                uint32
	MasterKeyFingerprint uint32
}

type cachedKey struct {
	key btcec.PrivateKey
}

func (c *cachedKey) Size() (uint64, error) {
	return 1, nil
}

type ScopedKeyManager struct {
	scope          KeyScope
	addrSchema     ScopeAddrSchema
	rootManager    *Manager
	addrs          map[addrKey]ManagedAddress
	acctInfo       map[uint32]*accountInfo
	deriveOnUnlock []*unlockDeriveInfo
	privKeyCache   *lru.Cache[DerivationPath, *cachedKey]

	mtx sync.RWMutex
}

func (s *ScopedKeyManager) AddrSchema() ScopeAddrSchema {
	return s.addrSchema
}

func (s *ScopedKeyManager) Scope() KeyScope {
	return s.scope
}

func (s *ScopedKeyManager) Close() {
	fmt.Println("ScopedKeyManager::Close() was not implemented yet.")
}
