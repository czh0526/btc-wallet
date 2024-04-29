package waddrmgr

import (
	"fmt"
	"github.com/czh0526/btc-wallet/walletdb"
	"github.com/czh0526/btc-wallet/walletdb/migration"
)

var versions = []migration.Version{
	{
		Number:    2,
		Migration: upgradeToVersion2,
	},
	{
		Number:    5,
		Migration: upgradeToVersion5,
	},
	{
		Number:    6,
		Migration: populateBirthdayBlock,
	},
	{
		Number:    7,
		Migration: resetSyncedBlockToBirthday,
	},
	{
		Number:    8,
		Migration: storeMaxReorgDepth,
	},
}

func getLatestVersion() uint32 {
	return versions[len(versions)-1].Number
}

func upgradeToVersion2(ns walletdb.ReadWriteBucket) error {
	fmt.Println("upgradeToVersion2() has not been implemented yet")
	return nil
}

func upgradeToVersion5(ns walletdb.ReadWriteBucket) error {
	fmt.Println("upgradeToVersion5() has not been implemented yet")
	return nil
}

func populateBirthdayBlock(ns walletdb.ReadWriteBucket) error {
	fmt.Println("populateBirthdayBlock() has not been implemented yet")
	return nil
}

func resetSyncedBlockToBirthday(ns walletdb.ReadWriteBucket) error {
	fmt.Println("resetSyncedBlockToBirthday() has not been implemented yet")
	return nil
}

func storeMaxReorgDepth(ns walletdb.ReadWriteBucket) error {
	fmt.Println("storeMaxReorgDepth() has not been implemented yet")
	return nil
}
