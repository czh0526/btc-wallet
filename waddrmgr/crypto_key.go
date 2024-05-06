package waddrmgr

import "github.com/czh0526/btc-wallet/snacl"

type cryptoKey struct {
	snacl.CryptoKey
}

func (ck *cryptoKey) Bytes() []byte {
	return ck.CryptoKey[:]
}

func (ck *cryptoKey) CopyBytes(from []byte) {
	copy(ck.CryptoKey[:], from)
}

func defaultNewCryptoKey() (EncryptorDecryptor, error) {
	key, err := snacl.GenerateCryptoKey()
	if err != nil {
		return nil, err
	}
	return &cryptoKey{*key}, nil
}
