package waddrmgr

import (
	"crypto/sha256"
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

	masterHDPrivName    = []byte("mhdpriv")
	masterHDPubName     = []byte("mhdpub")
	masterPrivKeyName   = []byte("mpriv")
	masterPubKeyName    = []byte("mpub")
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

var (
	nullVal = []byte{0}
)

type accountType uint8

const (
	accountDefault   accountType = 0
	accountWatchOnly accountType = 1
)

type syncStatus uint8

const (
	ssNone    syncStatus = 0
	ssPartial syncStatus = 1
	ssFull    syncStatus = 2
)

type addressType uint8

const (
	adtChain         addressType = 0
	adtImport        addressType = 1
	adtScript        addressType = 2
	adtWitnessScript addressType = 3
	adtTaprootScript addressType = 4
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

type dbWatchOnlyAccountRow struct {
	dbAccountRow
	pubKeyEncrypted      []byte
	masterKeyFingerprint uint32
	nextExternalIndex    uint32
	nextInternalIndex    uint32
	name                 string
	addrSchema           *ScopeAddrSchema
}

func createManagerNS(ns walletdb.ReadWriteBucket,
	defaultScopes map[KeyScope]ScopeAddrSchema) error {

	mainBucket, err := ns.CreateBucket(mainBucketName)
	if err != nil {
		str := "failed to create main bucket"
		return managerError(ErrDatabase, str, err)
	}
	fmt.Printf("【 new ns 】=> %s => %s \n", ns.Name(), mainBucketName)

	_, err = ns.CreateBucket(syncBucketName)
	if err != nil {
		str := "failed to create sync bucket"
		return managerError(ErrDatabase, str, err)
	}
	fmt.Printf("【 new ns 】=> %s => %s \n", ns.Name(), syncBucketName)

	scopeBucket, err := ns.CreateBucket(scopeBucketName)
	if err != nil {
		str := "failed to create scope bucket"
		return managerError(ErrDatabase, str, err)
	}
	fmt.Printf("【 new ns 】=> %s => %s \n", ns.Name(), scopeBucketName)

	scopeSchemas, err := ns.CreateBucket(scopeSchemaBucketName)
	if err != nil {
		str := "failed to create scope schemas"
		return managerError(ErrDatabase, str, err)
	}
	fmt.Printf("【 new ns 】=> %s => %s \n", ns.Name(), scopeSchemaBucketName)

	for scope, scopeSchema := range defaultScopes {
		scope, scopeSchema := scope, scopeSchema
		//fmt.Printf("\t new scope => %v \n", scope)
		scopeKey := scopeToBytes(&scope)
		schemaBytes := scopeSchemaToBytes(&scopeSchema)
		err := scopeSchemas.Put(scopeKey[:], schemaBytes)
		if err != nil {
			return err
		}
		fmt.Printf("\t【 write `%s` 】%v -> %v \n", scopeSchemas.Name(), scopeKey, schemaBytes)

		err = createScopedManagerNS(scopeBucket, &scope)
		if err != nil {
			return err
		}

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
	fmt.Printf("【 put_create_date 】%s => %s: %s -> %v bytes\n", ns.Name(), mainBucket.Name(), mgrCreateDateName, len(dateBytes))
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
	fmt.Printf("\t【 write `%s` 】 %v => %s \n", ns.Name(), scope, acctBucketName)

	_, err = scopeBucket.CreateBucket(addrBucketName)
	if err != nil {
		str := "failed to create address bucket"
		return managerError(ErrDatabase, str, err)
	}
	fmt.Printf("\t【 write `%s` 】 %v => %s \n", ns.Name(), scope, addrBucketName)

	_, err = scopeBucket.CreateBucket(usedAddrBucketName)
	if err != nil {
		str := "failed to create used address bucket"
		return managerError(ErrDatabase, str, err)
	}
	fmt.Printf("\t【 write `%s` 】 %v => %s \n", ns.Name(), scope, usedAddrBucketName)

	_, err = scopeBucket.CreateBucket(addrAcctIdxBucketName)
	if err != nil {
		str := "failed to create address index bucket"
		return managerError(ErrDatabase, str, err)
	}
	fmt.Printf("\t【 write `%s` 】 %v => %s \n", ns.Name(), scope, addrAcctIdxBucketName)

	_, err = scopeBucket.CreateBucket(acctNameIdxBucketName)
	if err != nil {
		str := "failed to create account name index bucket"
		return managerError(ErrDatabase, str, err)
	}
	fmt.Printf("\t【 write `%s` 】 %v => %s \n", ns.Name(), scope, acctNameIdxBucketName)

	_, err = scopeBucket.CreateBucket(acctIDIdxBucketName)
	if err != nil {
		str := "failed to create an account id index bucket"
		return managerError(ErrDatabase, str, err)
	}
	fmt.Printf("\t【 write `%s` 】 %v => %s \n", ns.Name(), scope, acctIDIdxBucketName)

	_, err = scopeBucket.CreateBucket(metaBucketName)
	if err != nil {
		str := "failed to create meta bucket"
		return managerError(ErrDatabase, str, err)
	}
	fmt.Printf("\t【 write `%s` 】 %v => %s \n", ns.Name(), scope, metaBucketName)

	return nil

}

func fetchLastAccount(ns walletdb.ReadBucket, scope *KeyScope) (uint32, error) {
	scopedBucket, err := fetchReadScopeBucket(ns, scope)
	if err != nil {
		return 0, err
	}

	metaBucket := scopedBucket.NestedReadBucket(metaBucketName)

	val := metaBucket.Get(lastAccountName)
	if val == nil {
		return (1 << 32) - 1, nil
	}
	if len(val) != 4 {
		str := fmt.Sprintf("malformed metadata '%s' stored in database", lastAccountName)
		return 0, managerError(ErrDatabase, str, nil)
	}

	account := binary.LittleEndian.Uint32(val[0:4])
	return account, nil
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

func fetchCoinTypeKeys(ns walletdb.ReadWriteBucket, scope *KeyScope) ([]byte, []byte, error) {
	scopedBucket, err := fetchReadScopeBucket(ns, scope)
	if err != nil {
		return nil, nil, err
	}

	coinTypePubKeyEnc := scopedBucket.Get(coinTypePubKeyName)
	if coinTypePubKeyEnc == nil {
		str := "required encrypted cointype public key not stored in database"
		return nil, nil, managerError(ErrDatabase, str, nil)
	}

	coinTypePrivKeyEnc := scopedBucket.Get(coinTypePrivKeyName)
	if coinTypePrivKeyEnc == nil {
		str := "required encrypted cointype private key not stored in database"
		return nil, nil, managerError(ErrDatabase, str, nil)
	}

	return coinTypePubKeyEnc, coinTypePrivKeyEnc, nil
}

func putCoinTypeKeys(ns walletdb.ReadWriteBucket, scope *KeyScope,
	coinTypePubKeyEnc []byte, coinTypePrivKeyEnc []byte) error {

	scopedBucket, err := fetchWriteScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	if coinTypePubKeyEnc != nil {
		err := scopedBucket.Put(coinTypePubKeyName, coinTypePubKeyEnc)
		fmt.Printf("【 write `%s` 】%v: %s -> %v bytes \n",
			scopeBucketName, scope, coinTypePubKeyName, len(coinTypePubKeyEnc))
		if err != nil {
			str := "failed to store encryptrd cointype public key"
			return managerError(ErrDatabase, str, err)
		}
	}

	if coinTypePrivKeyEnc != nil {
		err := scopedBucket.Put(coinTypePrivKeyName, coinTypePrivKeyEnc)
		fmt.Printf("【 write `%s` 】%v: %s -> %v bytes \n",
			scopeBucketName, scope, coinTypePrivKeyName, len(coinTypePrivKeyEnc))
		if err != nil {
			str := "failed to store encryptrd cointype private key"
			return managerError(ErrDatabase, str, err)
		}
	}

	return nil
}

func putManagerVersion(ns walletdb.ReadWriteBucket, version uint32) error {
	bucket := ns.NestedReadWriteBucket(mainBucketName)

	verBytes := uint32ToBytes(version)
	err := bucket.Put(mgrVersionName, verBytes)
	fmt.Printf("【 put_version 】%s => %s: %s -> %v bytes \n", ns.Name(), mainBucketName, mgrVersionName, len(verBytes))
	if err != nil {
		str := "failed to store version"
		return managerError(ErrDatabase, str, err)
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

func fetchReadScopeBucket(ns walletdb.ReadBucket, scope *KeyScope) (walletdb.ReadBucket, error) {
	rootScopeBucket := ns.NestedReadBucket(scopeBucketName)

	scopeKey := scopeToBytes(scope)
	scopedBucket := rootScopeBucket.NestedReadBucket(scopeKey[:])
	if scopedBucket == nil {
		str := fmt.Sprintf("unable to find scope：%s: %v", ns.Name(), scope)
		return nil, managerError(ErrScopeNotFound, str, nil)
	}

	return scopedBucket, nil
}

func fetchWriteScopeBucket(ns walletdb.ReadWriteBucket, scope *KeyScope) (walletdb.ReadWriteBucket, error) {
	rootScopeBucket := ns.NestedReadWriteBucket(scopeBucketName)

	scopeKey := scopeToBytes(scope)
	scopedBucket := rootScopeBucket.NestedReadWriteBucket(scopeKey[:])
	if scopedBucket == nil {
		str := fmt.Sprintf("unable to find scope: %s: %v", ns.Name(), scope)
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

func fetchWatchingOnly(ns walletdb.ReadBucket) (bool, error) {
	bucket := ns.NestedReadBucket(mainBucketName)

	buf := bucket.Get(watchingOnlyName)
	if len(buf) != 1 {
		str := "malformed watching-only flag stored in database"
		return false, managerError(ErrDatabase, str, nil)
	}

	return buf[0] != 0, nil
}

func fetchMasterHDKeys(ns walletdb.ReadBucket) ([]byte, []byte) {
	bucket := ns.NestedReadBucket(mainBucketName)

	var masterHDPrivEnc, masterHDPubEnc []byte

	key := bucket.Get(masterHDPrivName)
	fmt.Printf("  【 read `%s` 】`%s` -> %v bytes \n", mainBucketName, masterHDPrivName, len(key))
	if key != nil {
		masterHDPrivEnc = make([]byte, len(key))
		copy(masterHDPrivEnc, key)
	}

	key = bucket.Get(masterHDPubName)
	fmt.Printf("  【 read `%s` 】`%s` -> %v bytes \n", mainBucketName, masterHDPubName, len(key))
	if key != nil {
		masterHDPubEnc = make([]byte, len(key))
		copy(masterHDPubEnc, key)
	}

	return masterHDPrivEnc, masterHDPubEnc
}

func putMasterHDKeys(ns walletdb.ReadWriteBucket, masterHDPrivEnc, masterHDPubEnc []byte) error {

	bucket := ns.NestedReadWriteBucket(mainBucketName)

	if masterHDPrivEnc != nil {
		err := bucket.Put(masterHDPrivName, masterHDPrivEnc)
		fmt.Printf("  【 write `%s` 】%s -> %v bytes \n", mainBucketName, masterHDPrivName, len(masterHDPrivEnc))
		if err != nil {
			str := "failed to store encrypted master HD private key"
			return managerError(ErrDatabase, str, err)
		}
	}

	if masterHDPubEnc != nil {
		err := bucket.Put(masterHDPubName, masterHDPubEnc)
		fmt.Printf("  【 write `%s` 】%s -> %v bytes \n", mainBucketName, masterHDPubName, len(masterHDPubEnc))
		if err != nil {
			str := "failed to store encrypted master HD public key"
			return managerError(ErrDatabase, str, err)
		}
	}

	return nil
}

func fetchMasterKeyParams(ns walletdb.ReadBucket) ([]byte, []byte, error) {
	bucket := ns.NestedReadBucket(mainBucketName)

	val := bucket.Get(masterPubKeyName)
	fmt.Printf("  【 read `%s` 】`%s` -> %v bytes \n", mainBucketName, masterPubKeyName, len(val))
	if val == nil {
		str := "required master public key parameters not stored in database"
		return nil, nil, managerError(ErrDatabase, str, nil)
	}
	pubParams := make([]byte, len(val))
	copy(pubParams, val)

	var privParams []byte
	val = bucket.Get(masterPrivKeyName)
	fmt.Printf("  【 read `%s` 】`%s` => %v bytes \n", mainBucketName, masterPrivKeyName, len(val))
	if val != nil {
		privParams = make([]byte, len(val))
		copy(privParams, val)
	}

	return pubParams, privParams, nil
}

func putMasterKeyParams(ns walletdb.ReadWriteBucket, pubParams, privParams []byte) error {
	bucket := ns.NestedReadWriteBucket(mainBucketName)

	if privParams != nil {
		err := bucket.Put(masterPrivKeyName, privParams)
		fmt.Printf("  【 write `%s` 】%s -> %v bytes \n", mainBucketName, masterPrivKeyName, len(privParams))
		if err != nil {
			str := "failed to store master private key parameters"
			return managerError(ErrDatabase, str, err)
		}
	}

	if pubParams != nil {
		err := bucket.Put(masterPubKeyName, pubParams)
		fmt.Printf("  【 write `%s` 】%s -> %v bytes \n", mainBucketName, masterPubKeyName, len(pubParams))
		if err != nil {
			str := "failed to store master public key parameters"
			return managerError(ErrDatabase, str, err)
		}
	}

	return nil
}

func fetchCryptoKeys(ns walletdb.ReadBucket) ([]byte, []byte, []byte, error) {
	bucket := ns.NestedReadBucket(mainBucketName)

	val := bucket.Get(cryptoPubKeyName)
	fmt.Printf("  【 read `%s` 】`%s` -> %v bytes \n", mainBucketName, cryptoPubKeyName, len(val))
	if val == nil {
		str := "required encrypted crypto public key parameters not stored in database"
		return nil, nil, nil, managerError(ErrDatabase, str, nil)
	}
	pubKey := make([]byte, len(val))
	copy(pubKey, val)

	var privKey []byte
	val = bucket.Get(cryptoPrivKeyName)
	fmt.Printf("  【 read `%s` 】`%s` -> %v bytes \n", mainBucketName, cryptoPrivKeyName, len(val))
	if val != nil {
		privKey = make([]byte, len(val))
		copy(privKey, val)
	}

	var scriptKey []byte
	val = bucket.Get(cryptoScriptKeyName)
	fmt.Printf("  【 read `%s` 】`%s` -> %v bytes \n", mainBucketName, cryptoScriptKeyName, len(val))
	if val != nil {
		scriptKey = make([]byte, len(val))
		copy(scriptKey, val)
	}

	return pubKey, privKey, scriptKey, nil
}

func putCryptoKeys(ns walletdb.ReadWriteBucket,
	pubKeyEncrypted, privKeyEncrypted, scriptKeyEncrypted []byte) error {

	bucket := ns.NestedReadWriteBucket(mainBucketName)

	if pubKeyEncrypted != nil {
		err := bucket.Put(cryptoPubKeyName, pubKeyEncrypted)
		fmt.Printf("  【 write `%s` 】%s -> %v bytes \n", mainBucketName, cryptoPubKeyName, len(pubKeyEncrypted))
		if err != nil {
			str := "failed to store encrypted crypto public key"
			return managerError(ErrDatabase, str, err)
		}
	}

	if privKeyEncrypted != nil {
		err := bucket.Put(cryptoPrivKeyName, privKeyEncrypted)
		fmt.Printf("  【 write `%s` 】%s -> %v bytes \n", mainBucketName, cryptoPrivKeyName, len(privKeyEncrypted))
		if err != nil {
			str := "failed to store encrypted crypto private key"
			return managerError(ErrDatabase, str, err)
		}
	}

	if scriptKeyEncrypted != nil {
		err := bucket.Put(cryptoScriptKeyName, scriptKeyEncrypted)
		fmt.Printf("  【 write `%s` 】%s -> %v bytes \n", mainBucketName, cryptoScriptKeyName, len(scriptKeyEncrypted))
		if err != nil {
			str := "failed to store encrypted crypto script key"
			return managerError(ErrDatabase, str, err)
		}
	}

	return nil
}

func fetchScopeAddrSchema(ns walletdb.ReadBucket,
	scope *KeyScope) (*ScopeAddrSchema, error) {

	schemaBucket := ns.NestedReadBucket(scopeSchemaBucketName)
	if schemaBucket == nil {
		str := "unable to find scope schema bucket"
		return nil, managerError(ErrScopeNotFound, str, nil)
	}

	scopeKey := scopeToBytes(scope)
	schemaBytes := schemaBucket.Get(scopeKey[:])
	fmt.Printf("【 read `%s` 】`%v` -> `%v` \n", scopeSchemaBucketName, scope, schemaBytes)
	if schemaBytes == nil {
		str := fmt.Sprintf("unable to find scope, %s: %v", ns.Name(), scope)
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

func forEachAccountAddress(ns walletdb.ReadBucket, scope *KeyScope,
	account uint32, fn func(rowInterface interface{}) error) error {

	scopedBucket, err := fetchReadScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	bucket := scopedBucket.NestedReadBucket(addrAcctIdxBucketName).
		NestedReadBucket(uint32ToBytes(account))
	if bucket == nil {
		return nil
	}

	err = bucket.ForEach(func(k, v []byte) error {
		if v == nil {
			return nil
		}

		addrRow, err := fetchAddressByHash(ns, scope, k)
		if err != nil {
			if merr, ok := err.(*ManagerError); ok {
				desc := fmt.Sprintf("failed to fetch address hash:'%s':  %v",
					k, merr.Description)
				merr.Description = desc
				return merr
			}
			return err
		}

		return fn(addrRow)
	})
	if err != nil {
		return maybeConvertDbError(err)
	}
	return nil
}

type dbAddressRow struct {
	addrType   addressType
	account    uint32
	addTime    uint64
	syncStatus syncStatus
	rawData    []byte
}

type dbChainAddressRow struct {
	dbAddressRow
	branch uint32
	index  uint32
}

func serializeChainedAddress(branch, index uint32) []byte {
	rawData := make([]byte, 0)
	binary.LittleEndian.PutUint32(rawData[0:4], branch)
	binary.LittleEndian.PutUint32(rawData[4:8], index)
	return rawData
}

func deserializeChainedAddress(row *dbAddressRow) (*dbChainAddressRow, error) {
	if len(row.rawData) != 8 {
		str := "malformed serialized chained address"
		return nil, managerError(ErrDatabase, str, nil)
	}

	retRow := dbChainAddressRow{
		dbAddressRow: *row,
	}

	retRow.branch = binary.LittleEndian.Uint32(row.rawData[0:4])
	retRow.index = binary.LittleEndian.Uint32(row.rawData[4:8])
	return &retRow, nil
}

type dbImportedAddressRow struct {
	dbAddressRow
	encryptedPubKey  []byte
	encryptedPrivKey []byte
}

func serializeImportedAddress(encryptedPubKey, encryptedPrivKey []byte) []byte {
	pubLen := uint32(len(encryptedPubKey))
	privLen := uint32(len(encryptedPrivKey))
	rawData := make([]byte, 8+pubLen+privLen)

	binary.LittleEndian.PutUint32(rawData[0:4], pubLen)
	copy(rawData[4:4+pubLen], encryptedPubKey)

	offset := 4 + pubLen
	binary.LittleEndian.PutUint32(rawData[offset:offset+4], privLen)
	offset += 4
	copy(rawData[offset:offset+privLen], encryptedPrivKey)
	return rawData
}

func deserializeImportedAddress(row *dbAddressRow) (*dbImportedAddressRow, error) {
	if len(row.rawData) < 8 {
		str := "malformed serialized imported address"
		return nil, managerError(ErrDatabase, str, nil)
	}

	retRow := dbImportedAddressRow{
		dbAddressRow: *row,
	}

	pubLen := binary.LittleEndian.Uint32(row.rawData[0:4])
	retRow.encryptedPubKey = make([]byte, pubLen)
	copy(retRow.encryptedPubKey, row.rawData[4:4+pubLen])
	offset := 4 + pubLen
	privLen := binary.LittleEndian.Uint32(row.rawData[offset : offset+4])
	offset += 4
	retRow.encryptedPrivKey = make([]byte, privLen)
	copy(retRow.encryptedPrivKey, row.rawData[offset:offset+privLen])

	return &retRow, nil
}

type dbScriptAddressRow struct {
	dbAddressRow
	encryptedHash   []byte
	encryptedScript []byte
}

func serializeScriptAddress(encryptedHash, encryptedScript []byte) []byte {
	hashLen := uint32(len(encryptedHash))
	scriptLen := uint32(len(encryptedScript))
	rawData := make([]byte, 8+hashLen+scriptLen)

	binary.LittleEndian.PutUint32(rawData[0:4], hashLen)
	copy(rawData[4:4+hashLen], encryptedHash)

	offset := 4 + hashLen
	binary.LittleEndian.PutUint32(rawData[offset:offset+4], scriptLen)
	offset += 4
	copy(rawData[offset:offset+scriptLen], encryptedScript)
	return rawData

}

func deserializeScriptAddress(row *dbAddressRow) (*dbScriptAddressRow, error) {
	if len(row.rawData) < 8 {
		str := "malformed serialized script address"
		return nil, managerError(ErrDatabase, str, nil)
	}

	retRow := dbScriptAddressRow{
		dbAddressRow: *row,
	}

	hashLen := binary.LittleEndian.Uint32(row.rawData[0:4])
	retRow.encryptedHash = make([]byte, hashLen)
	copy(retRow.encryptedHash, row.rawData[4:4+hashLen])

	offset := 4 + hashLen
	scriptLen := binary.LittleEndian.Uint32(row.rawData[offset : offset+4])
	offset += 4
	retRow.encryptedScript = make([]byte, scriptLen)
	copy(retRow.encryptedScript, row.rawData[offset:offset+scriptLen])

	return &retRow, nil
}

type dbWitnessScriptAddressRow struct {
	dbAddressRow
	witnessVersion  byte
	isSecretScript  bool
	encryptedHash   []byte
	encryptedScript []byte
}

func serializeWitnessScriptAddress(witnessVersion uint8, isSecretScript bool,
	encryptedHash, encryptedScript []byte) []byte {

	hashLen := uint32(len(encryptedHash))
	scriptLen := uint32(len(encryptedScript))
	rawData := make([]byte, 10+hashLen+scriptLen)

	rawData[0] = witnessVersion
	if isSecretScript {
		rawData[1] = 1
	}
	binary.LittleEndian.PutUint32(rawData[2:6], hashLen)
	copy(rawData[6:6+hashLen], encryptedHash)

	offset := 6 + hashLen
	binary.LittleEndian.PutUint32(rawData[offset:offset+4], scriptLen)
	offset += 4
	copy(rawData[offset:offset+scriptLen], encryptedScript)

	return rawData
}

func deserializeWitnessScriptAddress(
	row *dbAddressRow) (*dbWitnessScriptAddressRow, error) {

	const minLength = 1 + 1 + 4 + 4

	if len(row.rawData) < minLength {
		str := "malformed serialized witness script address"
		return nil, managerError(ErrDatabase, str, nil)
	}

	retRow := dbWitnessScriptAddressRow{
		dbAddressRow:   *row,
		witnessVersion: row.rawData[0],
		isSecretScript: row.rawData[1] == 1,
	}

	hashLen := binary.LittleEndian.Uint32(row.rawData[2:6])
	retRow.encryptedHash = make([]byte, hashLen)
	copy(retRow.encryptedHash, row.rawData[6:6+hashLen])

	offset := 6 + hashLen
	scriptLen := binary.LittleEndian.Uint32(row.rawData[offset : offset+4])
	offset += 4
	retRow.encryptedScript = make([]byte, scriptLen)
	copy(retRow.encryptedScript, row.rawData[offset:offset+scriptLen])

	return &retRow, nil
}

func serializeAddressRow(row *dbAddressRow) []byte {
	rdlen := len(row.rawData)
	buf := make([]byte, 18+rdlen)
	buf[0] = byte(row.addrType)
	binary.LittleEndian.PutUint32(buf[1:5], row.account)
	binary.LittleEndian.PutUint64(buf[5:13], row.addTime)
	buf[13] = byte(row.syncStatus)
	binary.LittleEndian.PutUint32(buf[14:18], uint32(rdlen))
	copy(buf[18:18+rdlen], row.rawData)
	return buf
}

func deserializeAddressRow(serializeAddress []byte) (*dbAddressRow, error) {
	if len(serializeAddress) != 18 {
		str := "malformed serialized address"
		return nil, managerError(ErrDatabase, str, nil)
	}

	row := dbAddressRow{}
	row.addrType = addressType(serializeAddress[0])
	row.account = binary.LittleEndian.Uint32(serializeAddress[1:5])
	row.addTime = binary.LittleEndian.Uint64(serializeAddress[5:13])
	row.syncStatus = syncStatus(serializeAddress[13])
	rdlen := binary.LittleEndian.Uint32(serializeAddress[14:18])
	row.rawData = make([]byte, rdlen)
	copy(row.rawData, serializeAddress[18:18+rdlen])

	return &row, nil
}

func putAddrAccountIndex(ns walletdb.ReadWriteBucket, scope *KeyScope,
	account uint32, addrHash []byte) error {

	scopedBucket, err := fetchWriteScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	bucket := scopedBucket.NestedReadWriteBucket(addrAcctIdxBucketName)

	err = bucket.Put(addrHash, uint32ToBytes(account))
	if err != nil {
		return err
	}

	bucket, err = bucket.CreateBucketIfNotExists(uint32ToBytes(account))
	if err != nil {
		return err
	}

	err = bucket.Put(addrHash, nullVal)
	if err != nil {
		str := fmt.Sprintf("failed to store addr account index key: %s", addrHash)
		return managerError(ErrDatabase, str, err)
	}

	return nil
}

func putAddress(ns walletdb.ReadWriteBucket, scope *KeyScope,
	addressID []byte, row *dbAddressRow) error {

	scopedBucket, err := fetchWriteScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	bucket := scopedBucket.NestedReadWriteBucket(addrBucketName)
	addrHash := sha256.Sum256(addressID)
	err = bucket.Put(addrHash[:], serializeAddressRow(row))
	if err != nil {
		str := fmt.Sprintf("failed to store address %x", addressID)
		return managerError(ErrDatabase, str, err)
	}

	return putAddrAccountIndex(ns, scope, row.account, addrHash[:])
}

func putChainedAddress(ns walletdb.ReadWriteBucket, scope *KeyScope,
	addressID []byte, account uint32, status syncStatus,
	branch, index uint32, addrType addressType) error {

	scopedBucket, err := fetchWriteScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	addrRow := dbAddressRow{
		addrType:   addrType,
		account:    account,
		addTime:    uint64(time.Now().Unix()),
		syncStatus: status,
		rawData:    serializeChainedAddress(branch, index),
	}
	if err := putAddress(ns, scope, addressID, &addrRow); err != nil {
		return err
	}

	accountID := uint32ToBytes(account)
	bucket := scopedBucket.NestedReadWriteBucket(addrBucketName)
	serializedAccount := bucket.Get(accountID)

	row, err := deserializeAccountRow(accountID, serializedAccount)
	if err != nil {
		return err
	}

	switch row.acctType {
	case accountDefault:
		arow, err := deserializeDefaultAccountRow(accountID, row)
		if err != nil {
			return err
		}

		nextExternalIndex := arow.nextExternalIndex
		nextInternalIndex := arow.nextInternalIndex
		if branch == InternalBranch {
			nextInternalIndex = index + 1
		} else {
			nextExternalIndex = index + 1
		}

		row.rawData = serializeDefaultAccountRow(
			arow.pubKeyEncrypted, arow.privKeyEncrypted,
			nextExternalIndex, nextInternalIndex, arow.name)

	case accountWatchOnly:
		arow, err := deserializeWatchOnlyAccountRow(accountID, row)
		if err != nil {
			return err
		}

		nextExternalIndex := arow.nextExternalIndex
		nextInternalIndex := arow.nextInternalIndex
		if branch == InternalBranch {
			nextInternalIndex = index + 1
		} else {
			nextExternalIndex = index + 1
		}

		row.rawData, err = serializeWatchOnlyAccountRow(
			arow.pubKeyEncrypted, arow.masterKeyFingerprint,
			nextExternalIndex, nextInternalIndex, arow.name, arow.addrSchema)
		if err != nil {
			return err
		}
	}

	err = bucket.Put(accountID, serializeAccountRow(row))
	if err != nil {
		str := fmt.Sprintf("failed to update next index for account %x, account %d", addressID, account)
		return managerError(ErrDatabase, str, err)
	}

	return nil
}

func putScriptAddress(ns walletdb.ReadWriteBucket, scope *KeyScope,
	addressID []byte, account uint32, status syncStatus,
	encryptedHash, encryptedScript []byte) error {

	rawData := serializeScriptAddress(encryptedHash, encryptedScript)
	addrRow := dbAddressRow{
		addrType:   adtScript,
		account:    account,
		addTime:    uint64(time.Now().Unix()),
		syncStatus: status,
		rawData:    rawData,
	}

	if err := putAddress(ns, scope, addressID, &addrRow); err != nil {
		return err
	}

	return nil
}

func fetchAddress(ns walletdb.ReadBucket, scope *KeyScope,
	addressID []byte) (interface{}, error) {

	addrHash := sha256.Sum256(addressID)
	return fetchAddressByHash(ns, scope, addrHash[:])
}

func fetchAddressByHash(ns walletdb.ReadBucket, scope *KeyScope,
	addrHash []byte) (interface{}, error) {

	scopedBucket, err := fetchReadScopeBucket(ns, scope)
	if err != nil {
		return nil, err
	}

	bucket := scopedBucket.NestedReadBucket(addrBucketName)

	serializedRow := bucket.Get(addrHash)
	if serializedRow == nil {
		str := "address not found"
		return nil, managerError(ErrAddressNotFound, str, nil)
	}

	row, err := deserializeAddressRow(serializedRow)
	if err != nil {
		return nil, err
	}

	switch row.addrType {
	case adtChain:
		return deserializeChainedAddress(row)
	case adtImport:
		return deserializeImportedAddress(row)
	case adtScript:
		return deserializeScriptAddress(row)
	case adtWitnessScript:
		return deserializeWitnessScriptAddress(row)
	case adtTaprootScript:
		return deserializeWitnessScriptAddress(row)
	}

	str := fmt.Sprintf("unsupported address type '%d'", row.addrType)
	return nil, managerError(ErrDatabase, str, nil)
}

func fetchAddressUsed(ns walletdb.ReadBucket, scope *KeyScope,
	addressID []byte) bool {

	scopedBucket, err := fetchReadScopeBucket(ns, scope)
	if err != nil {
		return false
	}

	bucket := scopedBucket.NestedReadBucket(usedAddrBucketName)

	addrHash := sha256.Sum256(addressID)
	return bucket.Get(addrHash[:]) != nil
}

func markAddressUsed(ns walletdb.ReadWriteBucket, scope *KeyScope,
	addressIID []byte) error {

	scopedBucket, err := fetchWriteScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	bucket := scopedBucket.NestedReadWriteBucket(usedAddrBucketName)

	addrHash := sha256.Sum256(addressIID)
	val := bucket.Get(addrHash[:])
	if val != nil {
		return nil
	}

	err = bucket.Put(addrHash[:], []byte{0})
	if err != nil {
		str := fmt.Sprintf("failed to mark address used %x", addressIID)
		return managerError(ErrDatabase, str, nil)
	}

	return nil
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
