package prompt

import "github.com/btcsuite/btcd/btcutil/hdkeychain"

func PrivatePass() ([]byte, error) {
	return []byte("abc123"), nil
}

func PublicPass() ([]byte, error) {
	return []byte("public"), nil
}

func Seed() ([]byte, error) {
	return hdkeychain.GenerateSeed(hdkeychain.RecommendedSeedLen)
}
