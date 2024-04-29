/*
   Copyright (C) BABEC. All rights reserved.
   Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

   SPDX-License-Identifier: Apache-2.0
*/

package seed

import (
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
)

func NewSeed() ([]byte, error) {
	return hdkeychain.GenerateSeed(hdkeychain.RecommendedSeedLen)
}
