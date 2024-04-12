/*
   Copyright (C) BABEC. All rights reserved.
   Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

   SPDX-License-Identifier: Apache-2.0
*/

package original

import (
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"log"
)

func GenerateKeyPair() {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		log.Fatalf("generate private key failed, err = %v", err)
	}

	pubKey := privKey.PubKey()
	pubKeyCompressed := pubKey.SerializeCompressed()
	pubKeyUncompressed := pubKey.SerializeUncompressed()

	addressPubKeyHashCompressed, err := btcutil.NewAddressPubKeyHash(
		btcutil.Hash160(pubKeyCompressed), &chaincfg.MainNetParams)
	if err != nil {
		log.Fatalf("generate address from compresssed public key failed, err = %v", err)
	}
	addressPubKeyHashUncompressed, err := btcutil.NewAddressPubKeyHash(
		btcutil.Hash160(pubKeyUncompressed), &chaincfg.MainNetParams)
	if err != nil {
		log.Fatalf("generate address from uncompresssed public key failed, err = %v", err)
	}

	log.Printf("Private Key (HEX): %x", privKey.Serialize())
	log.Println("Public Key (Compressed, HEX):", addressPubKeyHashCompressed.EncodeAddress())
	log.Println("Public Key (Uncompressed, HEX):", addressPubKeyHashUncompressed.EncodeAddress())
}
