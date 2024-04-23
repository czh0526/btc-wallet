package snacl

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"golang.org/x/crypto/scrypt"
	"io"
	"runtime/debug"
)

const (
	KeySize = 32
)

var (
	prng = rand.Reader
)

var (
	ErrInvalidPassword = errors.New("invalid password")
)

type CryptoKey [KeySize]byte

type Parameters struct {
	Salt   [KeySize]byte
	Digest [sha256.Size]byte
	N      int
	R      int
	P      int
}

type SecretKey struct {
	Key        *CryptoKey
	Parameters Parameters
}

func NewSecretKey(password *[]byte, n, r, p int) (*SecretKey, error) {
	sk := SecretKey{
		Key: (*CryptoKey)(&[KeySize]byte{}),
	}

	// 填充参数
	sk.Parameters.N = n
	sk.Parameters.R = r
	sk.Parameters.P = p
	_, err := io.ReadFull(prng, sk.Parameters.Salt[:])
	if err != nil {
		return nil, err
	}

	// 根据{password}, 派生一个 Key
	err = sk.deriveKey(password)
	if err != nil {
		return nil, err
	}

	// 完善签名
	sk.Parameters.Digest = sha256.Sum256(sk.Key[:])

	return &sk, nil
}

func (sk *SecretKey) DeriveKey(password *[]byte) error {
	if err := sk.deriveKey(password); err != nil {
		return err
	}

	digest := sha256.Sum256(sk.Key[:])
	if constantTimeCompare(digest[:], sk.Parameters.Digest[:]) != 1 {
		return ErrInvalidPassword
	}

	return nil
}

func (sk *SecretKey) deriveKey(password *[]byte) error {
	key, err := scrypt.Key(
		*password,
		sk.Parameters.Salt[:],
		sk.Parameters.N,
		sk.Parameters.R,
		sk.Parameters.P,
		len(sk.Key))
	if err != nil {
		return err
	}

	copy(sk.Key[:], key)
	zeroBytes(key)

	debug.FreeOSMemory()
	return nil
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func constantTimeCompare(x, y []byte) int {
	if len(x) != len(y) {
		return 0
	}

	var v byte
	for i := 0; i < len(x); i++ {
		v |= x[i] ^ y[i]
	}

	return int((uint32(v^0) - 1) >> 31)
}
