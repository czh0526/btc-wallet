/*
   Copyright (C) BABEC. All rights reserved.
   Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

   SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/hex"
	"encoding/pem"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"log"
	"os"
)

func main() {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		log.Fatalf("generate private key failed, err = %v", err)
	}

	pubKey := privKey.PubKey()
	addressPubKeyCompressed, err := btcutil.NewAddressPubKey(
		pubKey.SerializeCompressed(), &chaincfg.MainNetParams)

	privKeyFile, err := os.Create("data/private_key.pem")
	if err != nil {
		log.Fatalf("create private key pem file failed, err = %v", err)
	}

	privKeyPem := pem.Block{
		Type:  "Bitcoin Private Key",
		Bytes: privKey.Serialize(),
	}

	if err := pem.Encode(privKeyFile, &privKeyPem); err != nil {
		log.Fatalf("write private key to pem file failed, err = %v", err)
	}

	addressFile, err := os.Create("data/wallet_address.txt")
	if err != nil {
		log.Fatalf("create address file failed, err = %v", err)
	}

	if _, err := addressFile.WriteString(addressPubKeyCompressed.EncodeAddress()); err != nil {
		log.Fatalf("write address to file failed, err = %v", err)
	}

	log.Printf("Private Key (saved to file) : %s", hex.EncodeToString(privKey.Serialize()))
	log.Println("Wallet Address (saved to file):", addressPubKeyCompressed.EncodeAddress())

}
