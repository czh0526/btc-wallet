package key

import (
	"fmt"
	"github.com/czh0526/btc-wallet/snacl"
	"github.com/czh0526/btc-wallet/waddrmgr"
	"github.com/stretchr/testify/assert"
	"testing"
)

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

}

func TestCryptoKey_Generate(t *testing.T) {
	cryptoKey, err := snacl.GenerateCryptoKey()
	assert.NoError(t, err)

	fmt.Printf("crypto key => %#v \n", cryptoKey)
}
