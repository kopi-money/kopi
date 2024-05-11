package upgrades

import (
	"github.com/kopi-money/kopi/app/upgrades/v0_3_3"
	"github.com/kopi-money/kopi/app/upgrades/v0_3_4"
)

func UpgradeHandlers() Upgrades {
	return Upgrades{
		{
			UpgradeName:          "v033",
			CreateUpgradeHandler: v0_3_3.CreateUpgradeHandler,
		},
		{
			UpgradeName:          "v034",
			CreateUpgradeHandler: v0_3_4.CreateUpgradeHandler,
		},
	}
}
