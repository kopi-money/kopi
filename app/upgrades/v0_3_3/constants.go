package v0_3_3

import "github.com/kopi-money/kopi/app/upgrades"

// UpgradeName defines the on-chain upgrade name for the Migaloo v3.0.2 upgrade.
// this upgrade includes the fix for pfm
const UpgradeName = "v033"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
}
