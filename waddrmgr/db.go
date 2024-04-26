package waddrmgr

import "github.com/czh0526/btc-wallet/walletdb"

var (
	mainBucketName = []byte("main")

	syncBucketName = []byte("sync")

	scopeBucketName = []byte("scope")

	scopeSchemaBucketName = []byte("scope-schema")
)

func createManagerNS(ns walletdb.ReadWriteBucket,
	defaultScopes map[KeyScope]ScopeAddrSchema) error {

	mainBucket, err := ns.CreateBucket(mainBucketName)
	if err != nil {
		str := "failed to create main bucket"
		return managerError(ErrDatabase, str, err)
	}

	_, err = ns.CreateBucket(syncBucketName)
	if err != nil {
		str := "failed to create sync bucket"
		return managerError(ErrDatabase, str, err)
	}

	scopeBucket, err := ns.CreateBucket(scopeBucketName)
	if err != nil {
		str := "failed to create scope bucket"
		return managerError(ErrDatabase, str, err)
	}

	scopeSchemas, err := ns.CreateBucket(scopeSchemaBucketName)
	if err != nil {
		str := "failed to create scope schemas"
		return managerError(ErrDatabase, str, err)
	}
}
