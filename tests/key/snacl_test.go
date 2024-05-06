package key

import (
	"fmt"
	"github.com/czh0526/btc-wallet/snacl"
	"github.com/czh0526/btc-wallet/waddrmgr"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCryptoKey_Generate(t *testing.T) {
	cryptoKey, err := snacl.GenerateCryptoKey()
	assert.NoError(t, err)

	fmt.Printf("crypto key => %#v \n", cryptoKey)
}

func TestSecretKey_New(t *testing.T) {

	// 获取加密配置
	config := waddrmgr.DefaultScryptOptions

	// 设置密码
	passphrase := []byte("test-passphrase")

	// 生成密钥
	secretKey, err := snacl.NewSecretKey(&passphrase, config.N, config.R, config.P)
	assert.NoError(t, err)

	fmt.Printf("secret key => %#v \n", secretKey)
}

func TestSecretKey_Derive(t *testing.T) {
	var err error
	var key *snacl.SecretKey
	var password = []byte("sikrit")

	// 构建一个 SecretKey
	key, err = snacl.NewSecretKey(&password, snacl.DefaultN, snacl.DefaultR, snacl.DefaultP)
	assert.NoError(t, err)

	// 将 Secret Key 序列化
	param := key.Marshal()

	// 使用新的 SecretKey 对象反序列化
	var sk snacl.SecretKey
	err = sk.Unmarshal(param)
	assert.NoError(t, err)

	// 使用相同的 password 派生一个新的 key
	err = sk.DeriveKey(&password)
	assert.NoError(t, err)

	assert.Equal(t, key.Key[:], sk.Key[:])
}

func TestSecretKey_Derive_Invalid(t *testing.T) {
	var err error
	var key *snacl.SecretKey
	var password = []byte("sikrit")

	// 构建一个 SecretKey
	key, err = snacl.NewSecretKey(&password, snacl.DefaultN, snacl.DefaultR, snacl.DefaultP)
	assert.NoError(t, err)

	// 将 Secret Key 序列化
	param := key.Marshal()

	// 使用新的 SecretKey 对象反序列化
	var sk snacl.SecretKey
	err = sk.Unmarshal(param)
	assert.NoError(t, err)

	// 使用不同的 password 派生一个新的 key
	newPasswd := []byte("wrong passwd")
	err = sk.DeriveKey(&newPasswd)
	assert.Equal(t, err, snacl.ErrInvalidPassword)
}

func TestSecretKey_Encrypt_Decrypt(t *testing.T) {
	var err error
	var blob []byte
	var message = []byte("This is a test message !")
	var password = []byte("password")
	var key *snacl.SecretKey

	// 构建一个 SecretKey
	key, err = snacl.NewSecretKey(&password, snacl.DefaultN, snacl.DefaultR, snacl.DefaultP)
	assert.NoError(t, err)

	// 使用 SecretKey 对消息进行加密
	blob, err = key.Encrypt(message)
	assert.NoError(t, err)

	// 使用 SecretKey 对消息进行解密
	decryptedMsg, err := key.Decrypt(blob)
	assert.NoError(t, err)

	assert.Equal(t, message, decryptedMsg)
}
