package waddrmgr

import (
	"encoding/binary"
	"fmt"
	"github.com/czh0526/btc-wallet/walletdb"
	"time"
)

var (
	mainBucketName = []byte("main")

	syncBucketName = []byte("sync")

	scopeBucketName = []byte("scope")

	scopeSchemaBucketName = []byte("scope-schema")

	usedAddrBucketName = []byte("usedaddrs")

	acctBucketName        = []byte("acct")
	acctNameIdxBucketName = []byte("acctnameidx")
	acctIDIdxBucketName   = []byte("acctididx")

	addrBucketName        = []byte("addr")
	addrAcctIdxBucketName = []byte("addracctidx")

	metaBucketName  = []byte("meta")
	lastAccountName = []byte("lastaccount")

	mgrVersionName    = []byte("mgrver")
	mgrCreateDateName = []byte("mgrcreated")

	coinTypePrivKeyName = []byte("ctpriv")
	coinTypePubKeyName  = []byte("ctpub")

	masterHDPrivName = []byte("mhdpriv")
	masterHDPubName  = []byte("mhdpub")

	masterPrivKeyName = []byte("mpriv")
	masterPubKeyName  = []byte("mpub")

	cryptoPrivKeyName   = []byte("cpriv")
	cryptoPubKeyName    = []byte("cpub")
	cryptoScriptKeyName = []byte("cscript")

	watchingOnlyName = []byte("watchonly")
	birthdayName     = []byte("birthday")
)

var (
	LatestMgrVersion = getLatestVersion()

	latestMgrVersion = LatestMgrVersion
)

type accountType uint8

const (
	accountDefault   accountType = 0
	accountWatchOnly accountType = 1
)

type dbAccountRow struct {
	acctType accountType
	rawData  []byte
}

type dbDefaultAccountRow struct {
	dbAccountRow
	pubKeyEncrypted   []byte
	privKeyEncrypted  []byte
	nextExternalIndex uint32
	nextInternalIndex uint32
	name              string
}

func createManagerNS(ns walletdb.ReadWriteBucket,
	defaultScopes map[KeyScope]ScopeAddrSchema) error {

	mainBucket, err := ns.CreateBucket(mainBucketName)
	if err != nil {
		str := "failed to create main bucket"
		return managerError(ErrDatabase, str, err)
	}
	fmt.Printf("\t=> %s => %s \n", ns.Name(), mainBucketName)

	_, err = ns.CreateBucket(syncBucketName)
	if err != nil {
		str := "failed to create sync bucket"
		return managerError(ErrDatabase, str, err)
	}
	fmt.Printf("\t=> %s => %s \n", ns.Name(), syncBucketName)

	scopeBucket, err := ns.CreateBucket(scopeBucketName)
	if err != nil {
		str := "failed to create scope bucket"
		return managerError(ErrDatabase, str, err)
	}
	fmt.Printf("\t=> %s => %s \n", ns.Name(), scopeBucketName)

	scopeSchemas, err := ns.CreateBucket(scopeSchemaBucketName)
	if err != nil {
		str := "failed to create scope schemas"
		return managerError(ErrDatabase, str, err)
	}
	fmt.Printf("\t=> %s => %s \n", ns.Name(), scopeSchemaBucketName)

	for scope, scopeSchema := range defaultScopes {
		scope, scopeSchema := scope, scopeSchema

		scopeKey := scopeToBytes(&scope)
		schemaBytes := scopeSchemaToBytes(&scopeSchema)
		err := scopeSchemas.Put(scopeKey[:], schemaBytes)
		if err != nil {
			return err
		}
		fmt.Printf("\t=> %s => %s: %s -> %v bytes \n", ns.Name(), scopeSchemas.Name(), scopeKey, len(schemaBytes))

		err = createScopedManagerNS(scopeBucket, &scope)
		if err != nil {
			return err
		}
		fmt.Printf("\t=> %s => %s => %v \n", ns.Name(), scopeBucket.Name(), scope)

		err = putLastAccount(ns, &scope, DefaultAccountNum)
		if err != nil {
			return err
		}
	}

	if err := putManagerVersion(ns, latestMgrVersion); err != nil {
		return err
	}

	createDate := uint64(time.Now().Unix())
	var dateBytes [8]byte
	binary.LittleEndian.PutUint64(dateBytes[:], createDate)
	err = mainBucket.Put(mgrCreateDateName, dateBytes[:])
	if err != nil {
		str := "failed to store database creation time"
		return managerError(ErrDatabase, str, err)
	}

	return nil
}

func createScopedManagerNS(ns walletdb.ReadWriteBucket, scope *KeyScope) error {
	scopeKey := scopeToBytes(scope)
	scopeBucket, err := ns.CreateBucket(scopeKey[:])
	if err != nil {
		str := "failed to create sync bucket"
		return managerError(ErrDatabase, str, err)
	}

	_, err = scopeBucket.CreateBucket(acctBucketName)
	if err != nil {
		str := "failed to create account bucket"
		return managerError(ErrDatabase, str, err)
	}

	_, err = scopeBucket.CreateBucket(addrBucketName)
	if err != nil {
		str := "failed to create address bucket"
		return managerError(ErrDatabase, str, err)
	}

	_, err = scopeBucket.CreateBucket(usedAddrBucketName)
	if err != nil {
		str := "failed to create used address bucket"
		return managerError(ErrDatabase, str, err)
	}

	_, err = scopeBucket.CreateBucket(addrAcctIdxBucketName)
	if err != nil {
		str := "failed to create address index bucket"
		return managerError(ErrDatabase, str, err)
	}

	_, err = scopeBucket.CreateBucket(acctNameIdxBucketName)
	if err != nil {
		str := "failed to create account name index bucket"
		return managerError(ErrDatabase, str, err)
	}

	_, err = scopeBucket.CreateBucket(acctIDIdxBucketName)
	if err != nil {
		str := "failed to create an account id index bucket"
		return managerError(ErrDatabase, str, err)
	}

	_, err = scopeBucket.CreateBucket(metaBucketName)
	if err != nil {
		str := "failed to create meta bucket"
		return managerError(ErrDatabase, str, err)
	}

	return nil

}

func putCoinTypeKeys(ns walletdb.ReadWriteBucket, scope *KeyScope,
	coinTypePubKeyEnc []byte, coinTypePrivKeyEnc []byte) error {

	scopedBucket, err := fetchWriteScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	if coinTypePubKeyEnc != nil {
		err := scopedBucket.Put(coinTypePubKeyName, coinTypePubKeyEnc)
		if err != nil {
			str := "failed to store encryptrd cointype public key"
			return managerError(ErrDatabase, str, err)
		}
	}

	if coinTypePrivKeyEnc != nil {
		err := scopedBucket.Put(coinTypePrivKeyName, coinTypePrivKeyEnc)
		if err != nil {
			str := "failed to store encryptrd cointype private key"
			return managerError(ErrDatabase, str, err)
		}
	}

	return nil
}

func putDefaultAccountInfo(ns walletdb.ReadWriteBucket, scope *KeyScope,
	account uint32, encryptedPubKey, encryptedPrivKey []byte,
	nextExternalIndex, nextInternalIndex uint32, name string) error {

	rawData := serializeDefaultAccountRow(
		encryptedPubKey, encryptedPrivKey,
		nextExternalIndex, nextInternalIndex, name)

	acctRow := dbAccountRow{
		acctType: accountDefault,
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
	if err != nil {
		str := fmt.Sprintf("failed to store account name index key %s", name)
		return managerError(ErrDatabase, str, err)
	}

	return nil
}

func putLastAccount(ns walletdb.ReadWriteBucket, scope *KeyScope,
	account uint32) error {

	scopedBucket, err := fetchWriteScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	bucket := scopedBucket.NestedReadWriteBucket(metaBucketName)

	err = bucket.Put(lastAccountName, uint32ToBytes(account))
	if err != nil {
		str := fmt.Sprintf("failed to update metadata '%s'", lastAccountName)
		return managerError(ErrDatabase, str, err)
	}

	return nil
}

func putManagerVersion(ns walletdb.ReadWriteBucket, version uint32) error {
	bucket := ns.NestedReadWriteBucket(mainBucketName)

	verBytes := uint32ToBytes(version)
	err := bucket.Put(mgrVersionName, verBytes)
	if err != nil {
		str := "failed to store version"
		return managerError(ErrDatabase, str, err)
	}

	return nil
}

func putMasterHDKeys(ns walletdb.ReadWriteBucket, masterHDPrivEnc, masterHDPubEnc []byte) error {

	bucket := ns.NestedReadWriteBucket(mainBucketName)

	if masterHDPrivEnc != nil {
		err := bucket.Put(masterHDPrivName, masterHDPrivEnc)
		if err != nil {
			str := "failed to store encrypted master HD private key"
			return managerError(ErrDatabase, str, err)
		}
	}

	if masterHDPubEnc != nil {
		err := bucket.Put(masterHDPubName, masterHDPubEnc)
		if err != nil {
			str := "failed to store encrypted master HD public key"
			return managerError(ErrDatabase, str, err)
		}
	}

	return nil
}

func putMasterKeyParams(ns walletdb.ReadWriteBucket, pubParams, privParams []byte) error {
	bucket := ns.NestedReadWriteBucket(mainBucketName)

	if privParams != nil {
		err := bucket.Put(masterPrivKeyName, privParams)
		if err != nil {
			str := "failed to store master private key parameters"
			return managerError(ErrDatabase, str, err)
		}
	}

	if pubParams != nil {
		err := bucket.Put(masterPubKeyName, pubParams)
		if err != nil {
			str := "failed to store master public key parameters"
			return managerError(ErrDatabase, str, err)
		}
	}

	return nil
}

func putCryptoKeys(ns walletdb.ReadWriteBucket,
	pubKeyEncrypted, privKeyEncrypted, scriptKeyEncrypted []byte) error {

	bucket := ns.NestedReadWriteBucket(mainBucketName)

	if pubKeyEncrypted != nil {
		err := bucket.Put(cryptoPubKeyName, pubKeyEncrypted)
		if err != nil {
			str := "failed to store encrypted crypto public key"
			return managerError(ErrDatabase, str, err)
		}
	}

	if privKeyEncrypted != nil {
		err := bucket.Put(cryptoPrivKeyName, privKeyEncrypted)
		if err != nil {
			str := "failed to store encrypted crypto private key"
			return managerError(ErrDatabase, str, err)
		}
	}

	if scriptKeyEncrypted != nil {
		err := bucket.Put(cryptoScriptKeyName, scriptKeyEncrypted)
		if err != nil {
			str := "failed to store encrypted crypto script key"
			return managerError(ErrDatabase, str, err)
		}
	}

	return nil
}

func putWatchingOnly(ns walletdb.ReadWriteBucket, watchingOnly bool) error {
	bucket := ns.NestedReadWriteBucket(mainBucketName)

	var encoded byte
	if watchingOnly {
		encoded = byte(1)
	}

	if err := bucket.Put(watchingOnlyName, []byte{encoded}); err != nil {
		str := "failed to store watching only flag"
		return managerError(ErrDatabase, str, err)
	}

	return nil
}

func putBirthday(ns walletdb.ReadWriteBucket, t time.Time) error {
	var birthdayTimestamp [8]byte
	binary.LittleEndian.PutUint64(birthdayTimestamp[:], uint64(t.Unix()))

	bucket := ns.NestedReadWriteBucket(syncBucketName)
	if err := bucket.Put(birthdayName, birthdayTimestamp[:]); err != nil {
		str := "failed to store birthday"
		return managerError(ErrDatabase, str, err)
	}

	return nil
}

func serializeAccountRow(row *dbAccountRow) []byte {
	rdlen := len(row.rawData)
	buf := make([]byte, rdlen+5)
	buf[0] = byte(row.acctType)
	binary.LittleEndian.PutUint32(buf[1:5], uint32(rdlen))
	copy(buf[5:5+rdlen], row.rawData)

	return buf
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

func fetchReadScopeBucket(ns walletdb.ReadWriteBucket, scope *KeyScope) (walletdb.ReadBucket, error) {
	rootScopeBucket := ns.NestedReadBucket(scopeBucketName)

	scopeKey := scopeToBytes(scope)
	scopedBucket := rootScopeBucket.NestedReadBucket(scopeKey[:])
	if scopedBucket == nil {
		str := fmt.Sprintf("unable to find scope %v", scope)
		return nil, managerError(ErrScopeNotFound, str, nil)
	}

	return scopedBucket, nil
}

func fetchWriteScopeBucket(ns walletdb.ReadWriteBucket, scope *KeyScope) (walletdb.ReadWriteBucket, error) {
	rootScopeBucket := ns.NestedReadWriteBucket(scopeBucketName)

	scopeKey := scopeToBytes(scope)
	scopedBucket := rootScopeBucket.NestedReadWriteBucket(scopeKey[:])
	if scopedBucket == nil {
		str := fmt.Sprintf("unable to find scope %v", scope)
		return nil, managerError(ErrScopeNotFound, str, nil)
	}

	return scopedBucket, nil
}

func fetchManagerVersion(ns walletdb.ReadBucket) (uint32, error) {
	mainBucket := ns.NestedReadBucket(mainBucketName)
	verBytes := mainBucket.Get(mgrVersionName)
	if verBytes == nil {
		str := "required version number not stored in database"
		return 0, managerError(ErrDatabase, str, nil)
	}

	version := binary.LittleEndian.Uint32(verBytes)
	return version, nil
}

func fetchMasterKeyParams(ns walletdb.ReadBucket) ([]byte, []byte, error) {
	bucket := ns.NestedReadBucket(mainBucketName)

	val := bucket.Get(masterPubKeyName)
	if val == nil {
		str := "required master public key parameters not stored in database"
		return nil, nil, managerError(ErrDatabase, str, nil)
	}
	pubParams := make([]byte, len(val))
	copy(pubParams, val)

	var privParams []byte
	val = bucket.Get(masterPrivKeyName)
	if val != nil {
		privParams = make([]byte, len(val))
		copy(privParams, val)
	}

	return pubParams, privParams, nil
}

func fetchWatchingOnly(ns walletdb.ReadBucket) (bool, error) {
	bucket := ns.NestedReadBucket(mainBucketName)

	buf := bucket.Get(watchingOnlyName)
	if len(buf) != 1 {
		str := "malformed watching-only flag stored in database"
		return false, managerError(ErrDatabase, str, nil)
	}

	return buf[0] != 0, nil
}

func fetchCryptoKeys(ns walletdb.ReadBucket) ([]byte, []byte, []byte, error) {
	bucket := ns.NestedReadBucket(mainBucketName)

	val := bucket.Get(cryptoPubKeyName)
	if val == nil {
		str := "required encrypted crypto public key parameters not stored in database"
		return nil, nil, nil, managerError(ErrDatabase, str, nil)
	}
	pubKey := make([]byte, len(val))
	copy(pubKey, val)

	var privKey []byte
	val = bucket.Get(cryptoPrivKeyName)
	if val != nil {
		privKey = make([]byte, len(val))
		copy(privKey, val)
	}

	var scriptKey []byte
	val = bucket.Get(cryptoScriptKeyName)
	if val != nil {
		scriptKey = make([]byte, len(val))
		copy(scriptKey, val)
	}

	return pubKey, privKey, scriptKey, nil
}

func fetchScopeAddrSchema(ns walletdb.ReadBucket,
	scope *KeyScope) (*ScopeAddrSchema, error) {

	schemaBucket := ns.NestedReadBucket(scopeBucketName)
	if schemaBucket == nil {
		str := "unable to find scope schema bucket"
		return nil, managerError(ErrScopeNotFound, str, nil)
	}

	scopeKey := scopeToBytes(scope)
	schemaBytes := schemaBucket.Get(scopeKey[:])
	if schemaBytes == nil {
		str := fmt.Sprintf("unable to find scope %v", scope)
		return nil, managerError(ErrScopeNotFound, str, nil)
	}

	return scopeSchemaFromBytes(schemaBytes), nil
}

func fetchBirthday(ns walletdb.ReadBucket) (time.Time, error) {
	var t time.Time

	bucket := ns.NestedReadBucket(syncBucketName)
	birthdayTimestamp := bucket.Get(birthdayName)
	if len(birthdayTimestamp) != 8 {
		str := "malformed birthday stored in database"
		return t, managerError(ErrDatabase, str, nil)
	}

	t = time.Unix(int64(binary.BigEndian.Uint64(birthdayTimestamp)), 0)
	return t, nil
}

func forEachKeyScope(ns walletdb.ReadBucket, fn func(KeyScope) error) error {
	bucket := ns.NestedReadBucket(scopeBucketName)

	return bucket.ForEach(func(k, v []byte) error {
		if len(k) != 8 {
			return nil
		}

		scope := KeyScope{
			Purpose: binary.LittleEndian.Uint32(k),
			Coin:    binary.LittleEndian.Uint32(k[4:]),
		}

		return fn(scope)
	})
}

const scopeKeySize = 8

func scopeToBytes(scope *KeyScope) [scopeKeySize]byte {
	var scopeBytes [scopeKeySize]byte
	binary.LittleEndian.PutUint32(scopeBytes[:], scope.Purpose)
	binary.LittleEndian.PutUint32(scopeBytes[4:], scope.Coin)

	return scopeBytes
}

func scopeSchemaToBytes(schema *ScopeAddrSchema) []byte {
	var schemaBytes [2]byte
	schemaBytes[0] = byte(schema.InternalAddrType)
	schemaBytes[1] = byte(schema.ExternalAddrType)

	return schemaBytes[:]
}

func scopeSchemaFromBytes(schemaBytes []byte) *ScopeAddrSchema {
	return &ScopeAddrSchema{
		InternalAddrType: AddressType(schemaBytes[0]),
		ExternalAddrType: AddressType(schemaBytes[1]),
	}
}

func uint32ToBytes(number uint32) []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, number)
	return buf
}

func stringToBytes(s string) []byte {
	size := len(s)
	buf := make([]byte, 4+size)
	copy(buf[0:4], uint32ToBytes(uint32(size)))
	copy(buf[4:4+size], s)
	return buf
}

func maybeConvertDbError(err error) error {
	if _, ok := err.(ManagerError); ok {
		return err
	}

	return managerError(ErrDatabase, err.Error(), err)
}
