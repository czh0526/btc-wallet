package waddrmgr

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

type ScopedKeyManager struct {
}

func (s *ScopedKeyManager) Close() {

}
