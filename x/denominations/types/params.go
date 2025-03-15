package types

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/kopi-money/kopi/constants"
	"gopkg.in/yaml.v2"
)

func createDefaultCollateralDenoms() []CollateralDenom {
	return []CollateralDenom{
		{
			DexDenom:   constants.BaseCurrency,
			Ltv:        math.LegacyNewDecWithPrec(5, 1),
			MaxDeposit: math.NewInt(1_000_000_000),
		},
	}
}

func createDefaultCAssets() []CAsset {
	return []CAsset{}
}

func createDefaultDexDenoms() []DexDenom {
	return []DexDenom{
		{
			Name:         constants.BaseCurrency,
			MinLiquidity: math.NewInt(1_000_000_000_000),
			MinOrderSize: math.NewInt(1_000_000),
			Exponent:     6,
		},
	}
}

func createDefaultKCoins() []KCoin {
	return []KCoin{}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return Params{
		CAssets:          createDefaultCAssets(),
		CollateralDenoms: createDefaultCollateralDenoms(),
		DexDenoms:        createDefaultDexDenoms(),
		KCoins:           createDefaultKCoins(),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateDexDenoms(p); err != nil {
		return fmt.Errorf("invalid dex denoms: %w", err)
	}

	if err := validateKCoins(p); err != nil {
		return fmt.Errorf("invalid kcoins: %w", err)
	}

	if err := validateCollateralDenoms(p); err != nil {
		return fmt.Errorf("invalid collateral denoms: %w", err)
	}

	if err := validateCAssets(p); err != nil {
		return fmt.Errorf("invalid c asset denoms: %w", err)
	}

	if err := validateArbitrageDenoms(p); err != nil {
		return fmt.Errorf("invalid arbitrage denoms: %w", err)
	}

	return nil
}

func validateArbitrageDenoms(p Params) error {
	seen := make(map[string]struct{})

	for _, arbitrageDenom := range p.StrategyDenoms.ArbitrageDenoms {
		if err := validateArbitrageDenom(p, arbitrageDenom); err != nil {
			return fmt.Errorf("error validating arbitrage denom %v: %w", arbitrageDenom.DexDenom, err)
		}

		if _, has := seen[arbitrageDenom.DexDenom]; has {
			return fmt.Errorf("duplicate arbitrage denom: %v", arbitrageDenom.DexDenom)
		}

		if _, has := seen[arbitrageDenom.KCoin]; has {
			return fmt.Errorf("duplicate arbitrage kCoin reference: %v", arbitrageDenom.KCoin)
		}

		if _, has := seen[arbitrageDenom.CAsset]; has {
			return fmt.Errorf("duplicate arbitrage cAsset reference: %v", arbitrageDenom.CAsset)
		}

		seen[arbitrageDenom.DexDenom] = struct{}{}
		seen[arbitrageDenom.CAsset] = struct{}{}
	}

	return nil
}

func validateArbitrageDenom(p Params, arbitrageDenom ArbitrageDenom) error {
	if arbitrageDenom.DexDenom == "" {
		return fmt.Errorf("must not have empty name")
	}

	if !hasDenom(p.DexDenoms, arbitrageDenom.DexDenom) {
		return fmt.Errorf("must be dex denom")
	}

	if !hasKCoin(p.KCoins, arbitrageDenom.KCoin) {
		return fmt.Errorf("referenced kCoin does not exist")
	}

	if !hasCAsset(p.CAssets, arbitrageDenom.CAsset) {
		return fmt.Errorf("referenced cAsset does not exist")
	}

	if arbitrageDenom.BuyTradeAmount.IsNil() {
		return fmt.Errorf("buy trade amount is nil")
	}

	if arbitrageDenom.SellThreshold.IsNil() {
		return fmt.Errorf("sell trade amount is nil")
	}

	if arbitrageDenom.BuyThreshold.IsNil() {
		return fmt.Errorf("buy threshold amount is nil")
	}

	if arbitrageDenom.SellThreshold.IsNil() {
		return fmt.Errorf("sell thresold amount is nil")
	}

	if arbitrageDenom.RedemptionFee.IsNil() {
		return fmt.Errorf("redemption fee is nil")
	}

	if arbitrageDenom.RedemptionFeeReserveShare.IsNil() {
		return fmt.Errorf("redemption fee reserve share is nil")
	}

	if arbitrageDenom.BuyTradeAmount.LTE(math.ZeroInt()) {
		return fmt.Errorf("buy trade amount must be larger than 0")
	}

	if arbitrageDenom.SellTradeAmount.LTE(math.ZeroInt()) {
		return fmt.Errorf("sell trade amount must be larger than 0")
	}

	if arbitrageDenom.SellThreshold.LT(math.LegacyOneDec()) {
		return fmt.Errorf("sell threshold must not be smaller than 1")
	}

	if arbitrageDenom.BuyThreshold.GT(math.LegacyOneDec()) {
		return fmt.Errorf("buy threshold must not be smaller than 1")
	}

	if arbitrageDenom.RedemptionFee.GT(math.LegacyOneDec()) {
		return fmt.Errorf("redemption fee must not be larger than 1")
	}

	if arbitrageDenom.RedemptionFee.IsNegative() {
		return fmt.Errorf("redemption fee must not be smaller than 0")
	}

	if arbitrageDenom.RedemptionFeeReserveShare.GT(math.LegacyOneDec()) {
		return fmt.Errorf("redemption fee reserve share must not be larger than 1")
	}

	if arbitrageDenom.RedemptionFeeReserveShare.IsNegative() {
		return fmt.Errorf("redemption fee reserve share must not be smaller than 0")
	}

	return nil
}

func validateKCoins(p Params) error {
	seen := make(map[string]struct{})

	for _, kCoin := range p.KCoins {
		if err := validateKCoin(p, kCoin); err != nil {
			return fmt.Errorf("error validating kCoin %v: %w", kCoin.DexDenom, err)
		}

		if _, has := seen[kCoin.DexDenom]; has {
			return fmt.Errorf("duplicate cAsset base denom")
		}

		seen[kCoin.DexDenom] = struct{}{}

		for _, referenceToken := range kCoin.References {
			if _, has := seen[referenceToken]; has {
				return fmt.Errorf("duplicate reference token")
			}

			seen[referenceToken] = struct{}{}
		}
	}

	return nil
}

func validateKCoin(p Params, kCoin KCoin) error {
	if !hasDenom(p.DexDenoms, kCoin.DexDenom) {
		return fmt.Errorf("kCoin is no dex denom")
	}

	if len(kCoin.References) == 0 {
		return fmt.Errorf("no reference denoms given")
	}

	for _, reference := range kCoin.References {
		if !hasDenom(p.DexDenoms, reference) {
			return fmt.Errorf("reference %v is no dex denom", reference)
		}

		if reference == kCoin.DexDenom {
			return fmt.Errorf("must not self reference")
		}
	}

	if kCoin.MaxSupply.IsNil() {
		return fmt.Errorf("max supply is nil")
	}

	if kCoin.MaxMintAmount.IsNil() {
		return fmt.Errorf("max mint amount is nil")
	}

	if kCoin.MaxBurnAmount.IsNil() {
		return fmt.Errorf("max burn amount is nil")
	}

	if kCoin.MaxSupply.LT(math.ZeroInt()) {
		return fmt.Errorf("max supply must not be smaller than 0")
	}

	if kCoin.MaxMintAmount.LT(math.ZeroInt()) {
		return fmt.Errorf("max mint amount must not be smaller than 0")
	}

	if kCoin.MaxBurnAmount.LT(math.ZeroInt()) {
		return fmt.Errorf("max burn amount must not be smaller than 0")
	}

	return nil
}

func validateCAssets(p Params) error {
	seen := make(map[string]struct{})

	for _, cAsset := range p.CAssets {
		if err := validateCAsset(p, cAsset); err != nil {
			return fmt.Errorf("error validating cAsset denom %v: %w", cAsset.DexDenom, err)
		}

		if _, has := seen[cAsset.DexDenom]; has {
			return fmt.Errorf("duplicate cAsset denom: %v", cAsset.DexDenom)
		}

		seen[cAsset.DexDenom] = struct{}{}
	}

	return nil
}

func validateCAsset(p Params, cAsset CAsset) error {
	if !hasDenom(p.DexDenoms, cAsset.BaseDexDenom) {
		return fmt.Errorf("cAsset's base denom (%v) not found in dex denoms", cAsset.BaseDexDenom)
	}

	if !hasDenom(p.DexDenoms, cAsset.DexDenom) {
		return fmt.Errorf("cAsset's denom not found in dex denoms")
	}

	if cAsset.DexFeeShare.IsNil() {
		cAsset.DexFeeShare = math.LegacyZeroDec()
	}

	if cAsset.DexFeeShare.IsNegative() {
		return fmt.Errorf("dex fee share must not be smaller than 0")
	}

	if cAsset.DexFeeShare.GT(math.LegacyOneDec()) {
		return fmt.Errorf("dex fee share must not be larger than 1")
	}

	if cAsset.BorrowLimit.IsNil() {
		cAsset.BorrowLimit = math.LegacyZeroDec()
	}

	if cAsset.BorrowLimit.GT(math.LegacyOneDec()) {
		return fmt.Errorf("borrow limit must not be larger than 1")
	}

	if cAsset.MinimumLoanSize.IsNil() {
		cAsset.MinimumLoanSize = math.ZeroInt()
	}

	if cAsset.MinimumLoanSize.LT(math.ZeroInt()) {
		return fmt.Errorf("minimum loan size must not be smaller than zero")
	}

	return nil
}

func validateCollateralDenoms(p Params) error {
	seen := make(map[string]struct{})

	for _, collateralDenom := range p.CollateralDenoms {
		if err := validateCollateralDenom(p, collateralDenom); err != nil {
			return fmt.Errorf("error validating collateral denom %v: %w", collateralDenom.DexDenom, err)
		}

		if _, has := seen[collateralDenom.DexDenom]; has {
			return fmt.Errorf("duplicate collateral denom: %v", collateralDenom.DexDenom)
		}
		seen[collateralDenom.DexDenom] = struct{}{}
	}

	return nil
}

func validateCollateralDenom(p Params, collateralDenom CollateralDenom) error {
	if collateralDenom.Ltv.IsNil() {
		return fmt.Errorf("ltv is nil")
	}

	if collateralDenom.MaxDeposit.IsNil() {
		return fmt.Errorf("max_deposit is nil")
	}

	if collateralDenom.Ltv.IsNegative() {
		return fmt.Errorf("ltv must not be smaller than 0")
	}

	if collateralDenom.Ltv.GT(math.LegacyOneDec()) {
		return fmt.Errorf("ltv must not be larger than 1")
	}

	if collateralDenom.MaxDeposit.LT(math.ZeroInt()) {
		return fmt.Errorf("max deposit must not be smaller than 0")
	}

	if !hasDenom(p.DexDenoms, collateralDenom.DexDenom) {
		return fmt.Errorf("collateral denom has to be dex denom")
	}

	return nil
}

func validateDexDenoms(p Params) error {
	seen := make(map[string]struct{})

	for _, dexDenom := range p.DexDenoms {
		if err := validateDexDenom(dexDenom); err != nil {
			return fmt.Errorf("error validating dex denom %v: %w", dexDenom.Name, err)
		}

		if _, has := seen[dexDenom.Name]; has {
			return fmt.Errorf("duplicate dex denom: %v", dexDenom.Name)
		}
		seen[dexDenom.Name] = struct{}{}
	}

	return nil
}

func validateDexDenom(dexDenom DexDenom) error {
	if dexDenom.Name == "" {
		return fmt.Errorf("dex denom name cannot be empty")
	}

	if dexDenom.MinOrderSize.IsNil() {
		return fmt.Errorf("min order size is nil")
	}

	if dexDenom.MinOrderSize.LTE(math.ZeroInt()) {
		return fmt.Errorf("minimum order size has to be bigger than zero")
	}

	if dexDenom.Name != constants.BaseCurrency {
		if dexDenom.MinLiquidity.IsNil() {
			return fmt.Errorf("min liquidity is nil")
		}

		if dexDenom.MinLiquidity.LTE(math.ZeroInt()) {
			return fmt.Errorf("minimum liquidty must not be smaller than zero")
		}

		if dexDenom.ExtraVirtualLiquidity != nil {
			if dexDenom.ExtraVirtualLiquidity.IsNil() {
				return fmt.Errorf("min virtual liquidity is nil")
			}

			if dexDenom.ExtraVirtualLiquidity.LTE(math.ZeroInt()) {
				return fmt.Errorf("minimum virtual liquidty must not be smaller than zero")
			}
		}
	}

	if dexDenom.Exponent < 1 {
		return fmt.Errorf("exponent has to be at leat 1, was: %v", dexDenom.Exponent)
	}

	return nil
}

func hasDenom(dexDenoms []DexDenom, denom string) bool {
	for _, dexDenom := range dexDenoms {
		if dexDenom.Name == denom {
			return true
		}
	}

	return false
}

func hasKCoin(kcoins []KCoin, denom string) bool {
	for _, kCoin := range kcoins {
		if kCoin.DexDenom == denom {
			return true
		}
	}

	return false
}

func hasCAsset(cAssets []CAsset, denom string) bool {
	for _, cAsset := range cAssets {
		if cAsset.DexDenom == denom {
			return true
		}
	}

	return false
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}
