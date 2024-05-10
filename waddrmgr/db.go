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

		scopeKey := scopeToBytes(&scope)
		schemaBytes := scopeSchemaToBytes(&scopeSchema)
		err := scopeSchemas.Put(scopeKey[:], schemaBytes)
		if err != nil {
			return err
		}
		fmt.Printf("【 write db 】%s => %s: %v -> %v bytes \n", ns.Name(), scopeSchemas.Name(), scopeKey, len(schemaBytes))

		err = createScopedManagerNS(scopeBucket, &scope)
		if err != nil {
			return err
		}
		fmt.Printf("【 new ns 】%s => %s => %v \n", ns.Name(), scopeBucket.Name(), scope)

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
	fmt.Printf("【 write db 】%s => %s: %s -> %v bytes\n", ns.Name(), mainBucket.Name(), mgrCreateDateName, len(dateBytes))
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
		fmt.Printf("【 write db 】%s => %s => %v: %s -> %v bytes \n",
			ns.Name(), scopeBucketName, scopedBucket.Name(), coinTypePubKeyName, len(coinTypePubKeyEnc))
		if err != nil {
			str := "failed to store encryptrd cointype public key"
			return managerError(ErrDatabase, str, err)
		}
	}

	if coinTypePrivKeyEnc != nil {
		err := scopedBucket.Put(coinTypePrivKeyName, coinTypePrivKeyEnc)
		fmt.Printf("【 write db 】%s => %s => %v: %s -> %v bytes \n",
			ns.Name(), scopeBucketName, scopedBucket.Name(), coinTypePrivKeyName, len(coinTypePrivKeyEnc))
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
	fmt.Printf("【 write db 】%s => %s: %s -> %v bytes \n", ns.Name(), mainBucketName, mgrVersionName, len(verBytes))
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
	if key != nil {
		masterHDPrivEnc = make([]byte, len(key))
		copy(masterHDPrivEnc, key)
	}

	key = bucket.Get(masterHDPubName)
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
		fmt.Printf("【 write db 】%s => %s: %s => %v bytes \n",
			ns.Name(), mainBucketName, masterHDPrivName, len(masterHDPrivEnc))
		if err != nil {
			str := "failed to store encrypted master HD private key"
			return managerError(ErrDatabase, str, err)
		}
	}

	if masterHDPubEnc != nil {
		err := bucket.Put(masterHDPubName, masterHDPubEnc)
		fmt.Printf("【 write db 】%s => %s: %s => %v bytes \n",
			ns.Name(), mainBucketName, masterHDPubName, len(masterHDPubEnc))
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
	fmt.Printf("【 read db 】%s => %s: %s => %v bytes \n",
		ns.Name(), mainBucketName, masterPubKeyName, len(val))
	if val == nil {
		str := "required master public key parameters not stored in database"
		return nil, nil, managerError(ErrDatabase, str, nil)
	}
	pubParams := make([]byte, len(val))
	copy(pubParams, val)

	var privParams []byte
	val = bucket.Get(masterPrivKeyName)
	fmt.Printf("【 read db 】%s => %s: %s => %v bytes \n",
		ns.Name(), mainBucketName, masterPrivKeyName, len(val))
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
		fmt.Printf("【 write db 】%s => %s: %s => %v bytes \n",
			ns.Name(), mainBucketName, masterPrivKeyName, len(privParams))
		if err != nil {
			str := "failed to store master private key parameters"
			return managerError(ErrDatabase, str, err)
		}
	}

	if pubParams != nil {
		err := bucket.Put(masterPubKeyName, pubParams)
		fmt.Printf("【 write db 】%s => %s: %s => %v bytes \n",
			ns.Name(), mainBucketName, masterPubKeyName, len(pubParams))
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
	fmt.Printf("【 read db 】%s => %s: %s => %v bytes \n",
		ns.Name(), mainBucketName, cryptoPubKeyName, len(val))
	if val == nil {
		str := "required encrypted crypto public key parameters not stored in database"
		return nil, nil, nil, managerError(ErrDatabase, str, nil)
	}
	pubKey := make([]byte, len(val))
	copy(pubKey, val)

	var privKey []byte
	val = bucket.Get(cryptoPrivKeyName)
	fmt.Printf("【 read db 】%s => %s: %s => %v bytes \n",
		ns.Name(), mainBucketName, cryptoPrivKeyName, len(val))
	if val != nil {
		privKey = make([]byte, len(val))
		copy(privKey, val)
	}

	var scriptKey []byte
	val = bucket.Get(cryptoScriptKeyName)
	fmt.Printf("【 read db 】%s => %s: %s => %v bytes \n",
		ns.Name(), mainBucketName, cryptoScriptKeyName, len(val))
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
		fmt.Printf("【 write db 】%s => %s: %s => %v bytes \n",
			ns.Name(), mainBucketName, cryptoPubKeyName, len(pubKeyEncrypted))
		if err != nil {
			str := "failed to store encrypted crypto public key"
			return managerError(ErrDatabase, str, err)
		}
	}

	if privKeyEncrypted != nil {
		err := bucket.Put(cryptoPrivKeyName, privKeyEncrypted)
		fmt.Printf("【 write db 】%s => %s: %s => %v bytes \n",
			ns.Name(), mainBucketName, cryptoPrivKeyName, len(privKeyEncrypted))
		if err != nil {
			str := "failed to store encrypted crypto private key"
			return managerError(ErrDatabase, str, err)
		}
	}

	if scriptKeyEncrypted != nil {
		err := bucket.Put(cryptoScriptKeyName, scriptKeyEncrypted)
		fmt.Printf("【 write db 】%s => %s: %s => %v bytes \n",
			ns.Name(), mainBucketName, cryptoScriptKeyName, len(scriptKeyEncrypted))
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
	fmt.Printf("【 read db 】%s => %s: %v => %v bytes \n",
		ns.Name(), scopeSchemaBucketName, scopeKey, len(schemaBytes))
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
