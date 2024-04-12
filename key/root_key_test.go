/*
   Copyright (C) BABEC. All rights reserved.
   Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

   SPDX-License-Identifier: Apache-2.0
*/

package key

import (
	"fmt"
	"github.com/czh0526/btc-wallet/seed"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewRootKey(t *testing.T) {
	seedBytes, err := seed.NewSeed()
	assert.Nil(t, err)

	rootKey, err := NewRootKey(seedBytes)
	assert.Nil(t, err)

	fmt.Printf("root key => %v \n", rootKey)
}
