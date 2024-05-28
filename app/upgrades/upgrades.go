package upgrades

import "github.com/kopi-money/kopi/app/upgrades/v0_4_1"

func UpgradeHandlers() Upgrades {
	return Upgrades{
		{
			UpgradeName:          "v0_4_1",
			CreateUpgradeHandler: v0_4_1.CreateUpgradeHandler,
		},
	}
}
