package waddrmgr

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/czh0526/btc-wallet/walletdb"
)

type accountInfo struct {
	acctName string
	acctType accountType

	acctKeyEncrypted []byte
	acctKeyPriv      *hdkeychain.ExtendedKey
	acctKeyPub       *hdkeychain.ExtendedKey

	nextExternalIndex uint32
	lastExternalAddr  ManagedAddress

	nextInternalIndex uint32
	lastInternalAddr  ManagedAddress

	addrSchema           *ScopeAddrSchema
	masterKeyFingerprint uint32
}

func putDefaultAccountInfo(ns walletdb.ReadWriteBucket,
	scope *KeyScope, account uint32,
	encryptedPubKey, encryptedPrivKey []byte,
	nextExternalIndex, nextInternalIndex uint32,
	name string) error {

	rawData := serializeDefaultAccountRow(
		encryptedPubKey, encryptedPrivKey,
		nextExternalIndex, nextInternalIndex,
		name)

	acctRow := dbAccountRow{
		acctType: accountDefault,
		rawData:  rawData,
	}
	return putAccountInfo(ns, scope, account, &acctRow, name)
}

func putWatchOnlyAccountInfo(ns walletdb.ReadWriteBucket,
	scope *KeyScope, account uint32, encryptedPubKey []byte,
	masterKeyFingerprint, nextExternalIndex, nextInternalIndex uint32,
	name string, addrSchema *ScopeAddrSchema) error {

	rawData, err := serializeWatchOnlyAccountRow(
		encryptedPubKey, masterKeyFingerprint,
		nextExternalIndex, nextInternalIndex, name, addrSchema)
	if err != nil {
		return err
	}

	acctRow := dbAccountRow{
		acctType: accountWatchOnly,
		rawData:  rawData,
	}

	return putAccountInfo(ns, scope, account, &acctRow, name)
}

func putAccountInfo(ns walletdb.ReadWriteBucket, scope *KeyScope,
	account uint32, acctRow *dbAccountRow, name string) error {

	if err := putAccountRow(ns, scope, account, acctRow); err != nil {
		return err
	}
	if err := putAccountIDIndex(ns, scope, account, name); err != nil {
		return err
	}

	return putAccountNameIndex(ns, scope, account, name)
}

func putAccountRow(ns walletdb.ReadWriteBucket, scope *KeyScope,
	account uint32, row *dbAccountRow) error {

	scopedBucket, err := fetchWriteScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	bucket := scopedBucket.NestedReadWriteBucket(acctBucketName)
	err = bucket.Put(uint32ToBytes(account), serializeAccountRow(row))
	fmt.Printf("【 write db 】%s => %s => %v => %s: %v -> {%v} \n",
		ns.Name(), scopeBucketName, scope, acctBucketName, account, "AccountRow")
	if err != nil {
		str := fmt.Sprintf("failed to store account %d", account)
		return managerError(ErrDatabase, str, err)
	}

	return nil
}

func putAccountIDIndex(ns walletdb.ReadWriteBucket, scope *KeyScope,
	account uint32, name string) error {

	scopedBucket, err := fetchWriteScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	bucket := scopedBucket.NestedReadWriteBucket(acctIDIdxBucketName)
	err = bucket.Put(uint32ToBytes(account), stringToBytes(name))
	fmt.Printf("【 write db 】%s => %s => %v => %s: %v -> {%v} \n",
		ns.Name(), scopeBucketName, scope, acctIDIdxBucketName, account, name)
	if err != nil {
		str := fmt.Sprintf("failed to store account id index key %s", name)
		return managerError(ErrDatabase, str, err)
	}

	return nil
}

func putAccountNameIndex(ns walletdb.ReadWriteBucket, scope *KeyScope,
	account uint32, name string) error {

	scopedBucket, err := fetchWriteScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	bucket := scopedBucket.NestedReadWriteBucket(acctNameIdxBucketName)
	err = bucket.Put(stringToBytes(name), uint32ToBytes(account))
	fmt.Printf("【 write db 】%s => %s => %v => %s: %v -> {%v} \n",
		ns.Name(), scopeBucketName, scope, acctNameIdxBucketName, name, account)
	if err != nil {
		str := fmt.Sprintf("failed to store account name index key %s", name)
		return managerError(ErrDatabase, str, err)
	}

	return nil
}

func fetchAccountByName(ns walletdb.ReadBucket, scope *KeyScope,
	name string) (uint32, error) {

	scopedBucket, err := fetchReadScopeBucket(ns, scope)
	if err != nil {
		return 0, err
	}

	idxBucket := scopedBucket.NestedReadBucket(acctNameIdxBucketName)

	val := idxBucket.Get(stringToBytes(name))
	if val == nil {
		str := fmt.Sprintf("account `%s` not found", name)
		return 0, managerError(ErrAccountNotFound, str, nil)
	}

	return binary.LittleEndian.Uint32(val), nil
}

func fetchAccountInfo(ns walletdb.ReadBucket, scope *KeyScope,
	account uint32) (interface{}, error) {

	scopedBucket, err := fetchReadScopeBucket(ns, scope)
	if err != nil {
		return nil, err
	}

	acctBucket := scopedBucket.NestedReadBucket(acctBucketName)
	accountID := uint32ToBytes(account)
	serializedRow := acctBucket.Get(accountID)
	if serializedRow == nil {
		str := fmt.Sprintf("account %d not found", account)
		return nil, managerError(ErrAccountNotFound, str, nil)
	}

	row, err := deserializeAccountRow(accountID, serializedRow)
	if err != nil {
		return nil, err
	}

	switch row.acctType {
	case accountDefault:
		return deserializeDefaultAccountRow(accountID, row)
	case accountWatchOnly:
		return deserializeWatchOnlyAccountRow(accountID, row)
	}

	str := fmt.Sprintf("unsupported account type `%d`", row.acctType)
	return nil, managerError(ErrAccountNotFound, str, nil)
}

func serializeAccountRow(row *dbAccountRow) []byte {
	rdlen := len(row.rawData)
	buf := make([]byte, rdlen+5)
	buf[0] = byte(row.acctType)
	binary.LittleEndian.PutUint32(buf[1:5], uint32(rdlen))
	copy(buf[5:5+rdlen], row.rawData)

	return buf
}

func deserializeAccountRow(accountID []byte, serializedAccount []byte) (*dbAccountRow, error) {
	if len(serializedAccount) < 5 {
		str := fmt.Sprintf("malformed serialzde account for key %x", accountID)
		return nil, managerError(ErrDatabase, str, nil)
	}

	row := dbAccountRow{}
	row.acctType = accountType(serializedAccount[0])
	rdlen := binary.LittleEndian.Uint32(serializedAccount[1:5])
	row.rawData = make([]byte, rdlen)
	copy(row.rawData, serializedAccount[5:5+rdlen])

	return &row, nil
}

func serializeDefaultAccountRow(
	encryptedPubKey, encryptedPrivKey []byte,
	nextExternalIndex, nextInternalIndex uint32, name string) []byte {

	pubLen := uint32(len(encryptedPubKey))
	privLen := uint32(len(encryptedPrivKey))
	nameLen := uint32(len(name))
	rawData := make([]byte, 20+pubLen+privLen+nameLen)
	binary.LittleEndian.PutUint32(rawData[0:4], pubLen)
	copy(rawData[4:4+pubLen], encryptedPubKey)
	offset := 4 + pubLen
	binary.LittleEndian.PutUint32(rawData[offset:offset+4], privLen)
	offset += 4
	copy(rawData[offset:offset+privLen], encryptedPrivKey)
	offset += privLen
	binary.LittleEndian.PutUint32(rawData[offset:offset+4], nextExternalIndex)
	offset += 4
	binary.LittleEndian.PutUint32(rawData[offset:offset+4], nextInternalIndex)
	offset += 4
	binary.LittleEndian.PutUint32(rawData[offset:offset+4], nameLen)
	offset += 4
	copy(rawData[offset:offset+nameLen], name)

	return rawData
}

func serializeWatchOnlyAccountRow(encryptedPubKey []byte,
	masterKeyFingerprint, nextExternalIndex, nextInternalIndex uint32,
	name string, addrSchema *ScopeAddrSchema) ([]byte, error) {

	pubLen := uint32(len(encryptedPubKey))
	nameLen := uint32(len(name))

	addrSchemaExists := addrSchema != nil
	var addrSchemaBytes []byte
	if addrSchemaExists {
		addrSchemaBytes = scopeSchemaToBytes(addrSchema)
	}

	bufLen := 21 + pubLen + nameLen + uint32(len(addrSchemaBytes))
	buf := bytes.NewBuffer(make([]byte, 0, bufLen))

	// pub key
	err := binary.Write(buf, binary.LittleEndian, pubLen)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.LittleEndian, encryptedPubKey)
	if err != nil {
		return nil, err
	}

	// finger print
	err = binary.Write(buf, binary.LittleEndian, masterKeyFingerprint)
	if err != nil {
		return nil, err
	}

	// external index
	err = binary.Write(buf, binary.LittleEndian, nextExternalIndex)
	if err != nil {
		return nil, err
	}

	// internal index
	err = binary.Write(buf, binary.LittleEndian, nextInternalIndex)
	if err != nil {
		return nil, err
	}

	// name
	err = binary.Write(buf, binary.LittleEndian, nameLen)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.LittleEndian, []byte(name))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, addrSchemaExists)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.LittleEndian, addrSchemaBytes)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func deserializeDefaultAccountRow(accountID []byte, row *dbAccountRow) (*dbDefaultAccountRow, error) {
	if len(row.rawData) < 20 {
		str := fmt.Sprintf("malformed serialzed bip0044 account for key %x", accountID)
		return nil, managerError(ErrDatabase, str, nil)
	}

	retRow := dbDefaultAccountRow{
		dbAccountRow: *row,
	}

	pubLen := binary.LittleEndian.Uint32(row.rawData[0:4])
	retRow.pubKeyEncrypted = make([]byte, pubLen)
	copy(retRow.pubKeyEncrypted, row.rawData[4:4+pubLen])
	offset := 4 + pubLen

	privLen := binary.LittleEndian.Uint32(row.rawData[offset : offset+4])
	offset += 4
	retRow.privKeyEncrypted = make([]byte, privLen)
	copy(retRow.privKeyEncrypted, row.rawData[offset:offset+privLen])
	offset += privLen

	retRow.nextExternalIndex = binary.LittleEndian.Uint32(row.rawData[offset : offset+4])
	offset += 4

	retRow.nextInternalIndex = binary.LittleEndian.Uint32(row.rawData[offset : offset+4])
	offset += 4

	nameLen := binary.LittleEndian.Uint32(row.rawData[offset : offset+4])
	offset += 4
	retRow.name = string(row.rawData[offset : offset+nameLen])

	return &retRow, nil
}

func deserializeWatchOnlyAccountRow(accountID []byte, row *dbAccountRow) (*dbWatchOnlyAccountRow, error) {

	if len(row.rawData) < 21 {
		str := fmt.Sprintf("malformed serialzed watch-only account for key %x", accountID)
		return nil, managerError(ErrDatabase, str, nil)
	}

	retRow := dbWatchOnlyAccountRow{
		dbAccountRow: *row,
	}
	r := bytes.NewReader(row.rawData[:])

	var pubLen uint32
	err := binary.Read(r, binary.LittleEndian, &pubLen)
	if err != nil {
		return nil, err
	}
	retRow.pubKeyEncrypted = make([]byte, pubLen)
	err = binary.Read(r, binary.LittleEndian, &retRow.pubKeyEncrypted)
	if err != nil {
		return nil, err
	}

	err = binary.Read(r, binary.LittleEndian, &retRow.nextExternalIndex)
	if err != nil {
		return nil, err
	}
	err = binary.Read(r, binary.LittleEndian, &retRow.nextInternalIndex)
	if err != nil {
		return nil, err
	}

	var nameLen uint32
	err = binary.Read(r, binary.LittleEndian, &nameLen)
	if err != nil {
		return nil, err
	}
	name := make([]byte, nameLen)
	err = binary.Read(r, binary.LittleEndian, &name)
	if err != nil {
		return nil, err
	}
	retRow.name = string(name)

	var addrSchemaExists bool
	err = binary.Read(r, binary.LittleEndian, &addrSchemaExists)
	if err != nil {
		return nil, err
	}
	if addrSchemaExists {
		var addrSchemaBytes [2]byte
		err = binary.Read(r, binary.LittleEndian, &addrSchemaBytes)
		if err != nil {
			return nil, err
		}
		retRow.addrSchema = scopeSchemaFromBytes(addrSchemaBytes[:])
	}

	return &retRow, nil
}
