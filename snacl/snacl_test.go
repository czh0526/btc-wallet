package snacl

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/scrypt"
	"testing"
)

var (
	password = []byte("sikrit")
	message  = []byte("this is a secret message of sorts")
	key      *SecretKey
	params   []byte
	blob     []byte
)

func TestNewScryptKeyTwoTimes(t *testing.T) {
	password = []byte("passphrase")
	salt1 := []byte("salt1")
	key1, err := scrypt.Key(password, salt1, DefaultN, DefaultR, DefaultP, 256)
	assert.NoError(t, err)

	key2, err := scrypt.Key(password, salt1, DefaultN, DefaultR, DefaultP, 256)
	assert.NoError(t, err)
	assert.Equal(t, key1, key2)
}

func TestNewSecretKey(t *testing.T) {
	var err error
	key, err = NewSecretKey(&password, DefaultN, DefaultR, DefaultP)
	assert.NoError(t, err)
}

func TestNewCryptoKey(t *testing.T) {
	var cryptoKey *CryptoKey
	var err error
	cryptoKey, err = GenerateCryptoKey()
	assert.NoError(t, err)

	fmt.Printf("crypto key => %v \n", cryptoKey)
}

func TestDeriveKey(t *testing.T) {
	key1, err := NewSecretKey(&password, DefaultN, DefaultR, DefaultP)
	assert.NoError(t, err)
	err = key1.DeriveKey(&password)
	assert.NoError(t, err)
}

func TestUnmarshalAndDeriveKey(t *testing.T) {
	key1, err := NewSecretKey(&password, DefaultN, DefaultR, DefaultP)
	assert.NoError(t, err)

	marshalled := key1.Marshal()

	var key2 SecretKey
	err = key2.Unmarshal(marshalled)
	assert.NoError(t, err)
	assert.Equal(t, DefaultN, key2.Parameters.N)
	assert.Equal(t, DefaultP, key2.Parameters.P)
	assert.Equal(t, DefaultR, key2.Parameters.R)

	err = key2.DeriveKey(&password)
	assert.NoError(t, err)
	assert.Equal(t, key1.Key, key2.Key)
}

func TestEncryptAndDecrypt(t *testing.T) {
	var err error
	key, err = NewSecretKey(&password, DefaultN, DefaultR, DefaultP)
	assert.NoError(t, err)

	blob, err = key.Encrypt(message)
	assert.NoError(t, err)

	decryptedMessage, err := key.Decrypt(blob)
	assert.NoError(t, err)

	assert.Equal(t, message, decryptedMessage)
}
