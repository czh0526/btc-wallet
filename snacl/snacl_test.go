package snacl

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/nacl/secretbox"
	"io"
	"testing"
)

func TestConstantTimeCompare(t *testing.T) {
	a := []byte("Hello World")
	b := []byte("Hello World")

	ret := constantTimeCompare(a, b)
	assert.Equal(t, 1, ret)
}

func TestCryptoKey_Seal(t *testing.T) {

	var nonce [NonceSize]byte
	_, err := io.ReadFull(prng, nonce[:])
	assert.Nil(t, err)

	cryptoKey, err := GenerateCryptoKey()
	assert.Nil(t, err)

	in := []byte("Hello World")
	blob := secretbox.Seal(nil, in, &nonce, (*[KeySize]byte)(cryptoKey))
	fmt.Printf("size: %v, blob: %x\n", len(blob), blob)

	in = []byte("Hello World")
	blob = secretbox.Seal(nil, in, &nonce, (*[KeySize]byte)(cryptoKey))
	fmt.Printf("size: %v, blob: %x\n", len(blob), blob)

	in = []byte("Hello World 1")
	blob = secretbox.Seal(nil, in, &nonce, (*[KeySize]byte)(cryptoKey))
	fmt.Printf("size: %v, blob: %x\n", len(blob), blob)

	in = []byte("Hello World 12345678")
	blob = secretbox.Seal(nil, in, &nonce, (*[KeySize]byte)(cryptoKey))
	fmt.Printf("size: %v, blob: %x\n", len(blob), blob)
}

func TestCryptoKey_Open(t *testing.T) {

	// 构建 CryptoKey
	cryptoKey, err := GenerateCryptoKey()
	assert.Nil(t, err)

	// 加密消息
	msg := []byte("Hello World")
	encryptedMsg, err := cryptoKey.Encrypt(msg)
	assert.Nil(t, err)

	// 恢复消息
	msg2, err := cryptoKey.Decrypt(encryptedMsg)
	assert.Nil(t, err)

	assert.Equal(t, msg, msg2)
}
