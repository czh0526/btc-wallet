package snacl

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"github.com/czh0526/btc-wallet/internal/zero"
	"golang.org/x/crypto/nacl/secretbox"
	"golang.org/x/crypto/scrypt"
	"io"
	"runtime/debug"
)

const (
	KeySize   = 32
	NonceSize = 24
)

var (
	prng = rand.Reader
)

var (
	ErrInvalidPassword = errors.New("invalid password")
	ErrMalformed       = errors.New("malformed data")
	ErrDecryptFailed   = errors.New("unable to decrypt")
)

type CryptoKey [KeySize]byte

func (ck *CryptoKey) Encrypt(in []byte) ([]byte, error) {
	var nonce [NonceSize]byte
	_, err := io.ReadFull(prng, nonce[:])
	if err != nil {
		return nil, err
	}
	blob := secretbox.Seal(nil, in, &nonce, (*[KeySize]byte)(ck))
	return append(nonce[:], blob...), nil
}

func (ck *CryptoKey) Decrypt(in []byte) ([]byte, error) {
	if len(in) < NonceSize {
		return nil, ErrMalformed
	}

	var nonce [NonceSize]byte
	copy(nonce[:], in[:NonceSize])
	blob := in[NonceSize:]

	opened, ok := secretbox.Open(nil, blob, &nonce, (*[KeySize]byte)(ck))
	if !ok {
		return nil, ErrDecryptFailed
	}

	return opened, nil
}

func (ck *CryptoKey) Zero() {
	zero.Bytea32((*[KeySize]byte)(ck))
}

func GenerateCryptoKey() (*CryptoKey, error) {
	var key CryptoKey
	_, err := io.ReadFull(prng, key[:])
	if err != nil {
		return nil, err
	}

	return &key, nil
}

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

func (sk *SecretKey) Encrypt(in []byte) ([]byte, error) {
	return sk.Key.Encrypt(in)
}

func (sk *SecretKey) Decrypt(in []byte) ([]byte, error) {
	return sk.Key.Decrypt(in)
}

func (sk *SecretKey) Marshal() []byte {
	params := &sk.Parameters

	marshalled := make([]byte, KeySize+sha256.Size+24)

	b := marshalled
	copy(b[:KeySize], params.Salt[:])
	b = b[KeySize:]
	copy(b[:sha256.Size], params.Digest[:])
	b = b[sha256.Size:]
	binary.LittleEndian.PutUint64(b[:8], uint64(params.N))
	b = b[8:]
	binary.LittleEndian.PutUint64(b[:8], uint64(params.R))
	b = b[8:]
	binary.LittleEndian.PutUint64(b[:8], uint64(params.P))

	return marshalled

}

func (sk *SecretKey) Zero() {
	sk.Key.Zero()
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
