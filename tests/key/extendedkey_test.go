package key

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/czh0526/btc-wallet/netparams"
	"github.com/stretchr/testify/assert"
	"testing"
)

var masterKey = []byte("Bitcoin seed")

func TestHmac512(t *testing.T) {
	var err error

	hdSeed, err := hdkeychain.GenerateSeed(hdkeychain.RecommendedSeedLen)
	assert.NoError(t, err)

	hmac1 := hmac.New(sha512.New, masterKey)
	_, err = hmac1.Write(hdSeed)
	assert.NoError(t, err)
	lr1 := hmac1.Sum(nil)

	hmac2 := hmac.New(sha512.New, masterKey)
	_, err = hmac2.Write(hdSeed)
	assert.NoError(t, err)
	lr2 := hmac2.Sum(nil)

	assert.Equal(t, 64, len(lr1))
	assert.Equal(t, 64, len(lr2))
	assert.Equal(t, lr1, lr2)
}

func TestNewMaster(t *testing.T) {

	// 获取网络配置参数
	netParams := netparams.MainNetParams
	params := netParams.Params
	//params := chaincfg.MainNetParams

	// 生成随机数
	hdSeed, err := hdkeychain.GenerateSeed(hdkeychain.RecommendedSeedLen)
	assert.NoError(t, err)

	// 生成主私钥
	masterKey, err := hdkeychain.NewMaster(hdSeed, params)
	assert.NoError(t, err)
	fmt.Printf("master key \t=> %v \n", masterKey)

	addr, err := masterKey.Address(params)
	assert.NoError(t, err)
	fmt.Printf("\t address => %v \n", addr)

	neuter, err := masterKey.Neuter()
	assert.NoError(t, err)
	fmt.Printf("\t neuter => %v \n", neuter)
}

func TestBIP0032Vectors(t *testing.T) {
	testVec1MasterHex := "000102030405060708090a0b0c0d0e0f"
	//testVec2MasterHex := "fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542"
	//testVec3MasterHex := "4b381541583be4423346c643850da4b320e46a87ae3d2a4e6da11eba819cd4acba45d239319ac14f863b8d5ab5a0d0c64d2e8a1e7d1457df2e5a3c51c73235be"
	hkStart := uint32(0x80000000)

	tests := []struct {
		name     string
		master   string
		path     []uint32
		wantPub  string
		wantPriv string
		net      *chaincfg.Params
	}{
		{
			name:     "test vector 1 chain m",
			master:   testVec1MasterHex,
			path:     []uint32{},
			wantPub:  "xpub661MyMwAqRbcFtXgS5sYJABqqG9YLmC4Q1Rdap9gSE8NqtwybGhePY2gZ29ESFjqJoCu1Rupje8YtGqsefD265TMg7usUDFdp6W1EGMcet8",
			wantPriv: "xprv9s21ZrQH143K3QTDL4LXw2F7HEK3wJUD2nW2nRk4stbPy6cq3jPPqjiChkVvvNKmPGJxWUtg6LnF5kejMRNNU3TGtRBeJgk33yuGBxrMPHi",
			net:      &chaincfg.MainNetParams,
		},
		{
			name:     "test vector 1 chain m/0H",
			master:   testVec1MasterHex,
			path:     []uint32{hkStart},
			wantPub:  "xpub68Gmy5EdvgibQVfPdqkBBCHxA5htiqg55crXYuXoQRKfDBFA1WEjWgP6LHhwBZeNK1VTsfTFUHCdrfp1bgwQ9xv5ski8PX9rL2dZXvgGDnw",
			wantPriv: "xprv9uHRZZhk6KAJC1avXpDAp4MDc3sQKNxDiPvvkX8Br5ngLNv1TxvUxt4cV1rGL5hj6KCesnDYUhd7oWgT11eZG7XnxHrnYeSvkzY7d2bhkJ7",
			net:      &chaincfg.MainNetParams,
		},
		{
			name:     "test vector 1 chain m/0H/1",
			master:   testVec1MasterHex,
			path:     []uint32{hkStart, 1},
			wantPub:  "xpub6ASuArnXKPbfEwhqN6e3mwBcDTgzisQN1wXN9BJcM47sSikHjJf3UFHKkNAWbWMiGj7Wf5uMash7SyYq527Hqck2AxYysAA7xmALppuCkwQ",
			wantPriv: "xprv9wTYmMFdV23N2TdNG573QoEsfRrWKQgWeibmLntzniatZvR9BmLnvSxqu53Kw1UmYPxLgboyZQaXwTCg8MSY3H2EU4pWcQDnRnrVA1xe8fs",
			net:      &chaincfg.MainNetParams,
		},
		{
			name:     "test vector 1 chain m/0H/1/2H",
			master:   testVec1MasterHex,
			path:     []uint32{hkStart, 1, hkStart + 2},
			wantPub:  "xpub6D4BDPcP2GT577Vvch3R8wDkScZWzQzMMUm3PWbmWvVJrZwQY4VUNgqFJPMM3No2dFDFGTsxxpG5uJh7n7epu4trkrX7x7DogT5Uv6fcLW5",
			wantPriv: "xprv9z4pot5VBttmtdRTWfWQmoH1taj2axGVzFqSb8C9xaxKymcFzXBDptWmT7FwuEzG3ryjH4ktypQSAewRiNMjANTtpgP4mLTj34bhnZX7UiM",
			net:      &chaincfg.MainNetParams,
		},
		{
			name:     "test vector 1 chain m/0H/1/2H/2",
			master:   testVec1MasterHex,
			path:     []uint32{hkStart, 1, hkStart + 2, 2},
			wantPub:  "xpub6FHa3pjLCk84BayeJxFW2SP4XRrFd1JYnxeLeU8EqN3vDfZmbqBqaGJAyiLjTAwm6ZLRQUMv1ZACTj37sR62cfN7fe5JnJ7dh8zL4fiyLHV",
			wantPriv: "xprvA2JDeKCSNNZky6uBCviVfJSKyQ1mDYahRjijr5idH2WwLsEd4Hsb2Tyh8RfQMuPh7f7RtyzTtdrbdqqsunu5Mm3wDvUAKRHSC34sJ7in334",
			net:      &chaincfg.MainNetParams,
		},
		{
			name:     "test vector 1 chain m/0H/1/2H/2/1000000000",
			master:   testVec1MasterHex,
			path:     []uint32{hkStart, 1, hkStart + 2, 2, 1000000000},
			wantPub:  "xpub6H1LXWLaKsWFhvm6RVpEL9P4KfRZSW7abD2ttkWP3SSQvnyA8FSVqNTEcYFgJS2UaFcxupHiYkro49S8yGasTvXEYBVPamhGW6cFJodrTHy",
			wantPriv: "xprvA41z7zogVVwxVSgdKUHDy1SKmdb533PjDz7J6N6mV6uS3ze1ai8FHa8kmHScGpWmj4WggLyQjgPie1rFSruoUihUZREPSL39UNdE3BBDu76",
			net:      &chaincfg.MainNetParams,
		},
	}

	for _, test := range tests {
		masterSeed, err := hex.DecodeString(test.master)
		assert.NoError(t, err)

		extKey, err := hdkeychain.NewMaster(masterSeed, test.net)
		assert.NoError(t, err)

		for _, childNum := range test.path {
			var err error
			extKey, err = extKey.Derive(childNum)
			assert.NoError(t, err)
		}

		if extKey.Depth() != uint8(len(test.path)) {
			t.Errorf("Depth of key %d should match fixture path: %v", extKey.Depth(), len(test.path))
			continue
		}

		// 检查私钥
		privStr := extKey.String()
		assert.Equal(t, test.wantPriv, privStr)

		// 检查公钥
		pubKey, err := extKey.Neuter()
		assert.NoError(t, err)
		assert.Equal(t, test.wantPub, pubKey.String())

		pubKey, err = pubKey.Neuter()
		assert.NoError(t, err)
		assert.Equal(t, test.wantPub, pubKey.String())
	}
}

func TestBase58(t *testing.T) {
	var msg []byte
	var encoded string

	data := make([]byte, 78)
	_, err := rand.Read(data)
	assert.NoError(t, err)

	msg = []byte{0x04, 0x88, 0xad, 0xe4}
	msg = append(msg, data...)
	encoded = base58.Encode(msg)
	fmt.Printf("%v => %v \n", msg, encoded)

	msg = []byte{0x04, 0x88, 0xb2, 0x1e}
	msg = append(msg, data...)
	encoded = base58.Encode(msg)
	fmt.Printf("%v => %v \n", msg, encoded)
}
