/*
   Copyright (C) BABEC. All rights reserved.
   Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

   SPDX-License-Identifier: Apache-2.0
*/

package key

import (
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
)

func NewRootKey(seed []byte) (*hdkeychain.ExtendedKey, error) {
	return hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
}
