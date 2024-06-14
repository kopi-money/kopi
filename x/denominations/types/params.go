package types

import (
	"fmt"

	"cosmossdk.io/math"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/kopi-money/kopi/utils"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var (
	KeyCAssets          = []byte("CAssets")
	KeyDenomCollaterals = []byte("DenomCollaterals")
	KeyDexDenoms        = []byte("DexDenoms")
	KeyKCoins           = []byte("KCoins")
)

var _ paramtypes.ParamSet = (*Params)(nil)

func createDefaultCollateralDenoms() []*CollateralDenom {
	return []*CollateralDenom{
		{
			Denom:      utils.BaseCurrency,
			Ltv:        math.LegacyNewDecWithPrec(5, 1),
			MaxDeposit: math.NewInt(1_000_000_000),
		},
		{
			Denom:      "uwusdc",
			Ltv:        math.LegacyNewDecWithPrec(9, 1),
			MaxDeposit: math.NewInt(1_000_000_000),
		},
		{
			Denom:      "ucwusdc",
			Ltv:        math.LegacyNewDecWithPrec(95, 2),
			MaxDeposit: math.NewInt(1_000_000_000),
		},
		{
			Denom:      "ukusd",
			Ltv:        math.LegacyNewDecWithPrec(9, 1),
			MaxDeposit: math.NewInt(1_000_000_000),
		},
		{
			Denom:      "uckusd",
			Ltv:        math.LegacyNewDecWithPrec(95, 2),
			MaxDeposit: math.NewInt(1_000_000_000),
		},
		{
			Denom:      "swbtc",
			Ltv:        math.LegacyNewDecWithPrec(8, 1),
			MaxDeposit: math.NewInt(1_000_000_000),
		},
		{
			Denom:      "skbtc",
			Ltv:        math.LegacyNewDecWithPrec(8, 1),
			MaxDeposit: math.NewInt(1_000_000_000),
		},
	}
}

func createDefaultCAssets() []*CAsset {
	return []*CAsset{
		{
			Name:            "uckusd",
			BaseDenom:       "ukusd",
			DexFeeShare:     math.LegacyNewDecWithPrec(5, 1),
			BorrowLimit:     math.LegacyNewDecWithPrec(99, 2),
			MinimumLoanSize: math.NewInt(1000),
		},
		{
			Name:            "ucwusdc",
			BaseDenom:       "uwusdc",
			DexFeeShare:     math.LegacyNewDecWithPrec(5, 1),
			BorrowLimit:     math.LegacyNewDecWithPrec(99, 2),
			MinimumLoanSize: math.NewInt(1000),
		},
		{
			Name:            "sckbtc",
			BaseDenom:       "skbtc",
			DexFeeShare:     math.LegacyNewDecWithPrec(5, 1),
			BorrowLimit:     math.LegacyNewDecWithPrec(99, 2),
			MinimumLoanSize: math.NewInt(1000),
		},
	}
}

func createDefaultDexDenoms() []*DexDenom {
	return []*DexDenom{
		{
			Name:         utils.BaseCurrency,
			MinLiquidity: math.NewInt(10_000),
			MinOrderSize: math.NewInt(1),
		},
		{
			Name:         "uwusdc",
			Factor:       decPtr(math.LegacyNewDecWithPrec(25, 2)),
			MinLiquidity: math.NewInt(10_000_000),
			MinOrderSize: math.NewInt(1),
		},
		{
			Name:         "uwusdt",
			Factor:       decPtr(math.LegacyNewDecWithPrec(25, 2)),
			MinLiquidity: math.NewInt(10_000_000),
			MinOrderSize: math.NewInt(1_000_000),
		},
		{
			Name:         "ukusd",
			Factor:       decPtr(math.LegacyNewDecWithPrec(25, 2)),
			MinLiquidity: math.NewInt(10_000_000),
			MinOrderSize: math.NewInt(1),
		},
		{
			Name:         "uckusd",
			Factor:       decPtr(math.LegacyNewDecWithPrec(25, 2)),
			MinLiquidity: math.NewInt(10_000_000),
			MinOrderSize: math.NewInt(1_000_000),
		},
		{
			Name:         "ucwusdc",
			Factor:       decPtr(math.LegacyNewDecWithPrec(25, 2)),
			MinLiquidity: math.NewInt(10_000_000),
			MinOrderSize: math.NewInt(1_000_000),
		},
		{
			Name:         "swbtc",
			Factor:       decPtr(math.LegacyNewDecWithPrec(1, 3)),
			MinLiquidity: math.NewInt(1_000),
			MinOrderSize: math.NewInt(1_000_000),
		},
		{
			Name:         "skbtc",
			Factor:       decPtr(math.LegacyNewDecWithPrec(1, 3)),
			MinLiquidity: math.NewInt(1_000),
			MinOrderSize: math.NewInt(1_000_000),
		},
		{
			Name:         "sckbtc",
			Factor:       decPtr(math.LegacyNewDecWithPrec(1, 3)),
			MinLiquidity: math.NewInt(1_000),
			MinOrderSize: math.NewInt(1_000_000),
		},
	}
}

func createDefaultKCoins() []*KCoin {
	return []*KCoin{
		{
			Denom:         "ukusd",
			References:    []string{"uwusdc", "uwusdt"},
			MaxSupply:     math.NewInt(1_000_000_000_000),
			MaxMintAmount: math.NewInt(1_000_000),
			MaxBurnAmount: math.NewInt(1_000_000),
		},
		{
			Denom:         "skbtc",
			References:    []string{"swbtc"},
			MaxSupply:     math.NewInt(100_000_000),
			MaxMintAmount: math.NewInt(10_000),
			MaxBurnAmount: math.NewInt(10_000),
		},
	}
}

func decPtr(dec math.LegacyDec) *math.LegacyDec {
	return &dec
}

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
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

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyKCoins, &p.KCoins, func(a any) error {
			return validateKCoins(a, p.DexDenoms)
		}),
		paramtypes.NewParamSetPair(KeyCAssets, &p.CAssets, func(a any) error {
			return validateCAssets(a, p.DexDenoms)
		}),
		paramtypes.NewParamSetPair(KeyDenomCollaterals, &p.CollateralDenoms, func(a any) error {
			return validateCollateralDenoms(a, p.DexDenoms)
		}),

		paramtypes.NewParamSetPair(KeyDexDenoms, &p.DexDenoms, validateDexDenoms),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateDexDenoms(p.DexDenoms); err != nil {
		return err
	}

	if err := validateKCoins(p.KCoins, p.DexDenoms); err != nil {
		return err
	}

	if err := validateCollateralDenoms(p.CollateralDenoms, p.DexDenoms); err != nil {
		return err
	}

	if err := validateCAssets(p.CAssets, p.DexDenoms); err != nil {
		return err
	}

	return nil
}

func validateKCoins(v any, dexDenoms []*DexDenom) error {
	kCoins, ok := v.([]*KCoin)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	seen := make(map[string]struct{})

	for index, kCoin := range kCoins {
		if kCoin == nil {
			return fmt.Errorf("kCoin #%v is nil", index)
		}

		if err := validateKCoin(dexDenoms, kCoin); err != nil {
			return errors.Wrap(err, fmt.Sprintf("error validating kCoin %v", kCoin.Denom))
		}

		if _, has := seen[kCoin.Denom]; has {
			return fmt.Errorf("duplicate cAsset base denom")
		}

		seen[kCoin.Denom] = struct{}{}

		for _, referenceToken := range kCoin.References {
			if _, has := seen[referenceToken]; has {
				return fmt.Errorf("duplicate reference token")
			}

			seen[referenceToken] = struct{}{}
		}
	}

	return nil
}

func validateKCoin(dexDenoms []*DexDenom, kCoin *KCoin) error {
	if !hasDenom(dexDenoms, kCoin.Denom) {
		return fmt.Errorf("kCoin is no dex denom")
	}

	if len(kCoin.References) == 0 {
		return fmt.Errorf("no reference denoms given")
	}

	for _, reference := range kCoin.References {
		if !hasDenom(dexDenoms, reference) {
			return fmt.Errorf("reference %v is no dex denom", reference)
		}

		if reference == kCoin.Denom {
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

func validateCAssets(v any, dexDenoms []*DexDenom) error {
	cAssets, ok := v.([]*CAsset)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	seen := make(map[string]struct{})

	for index, cAsset := range cAssets {
		if cAsset == nil {
			return fmt.Errorf("cAsset #%v is nil", index)
		}

		if err := validateCAsset(dexDenoms, cAsset); err != nil {
			return errors.Wrap(err, fmt.Sprintf("error validating cAssets %v", cAsset.Name))
		}

		if _, has := seen[cAsset.Name]; has {
			return fmt.Errorf("duplicate cAsset denom")
		}

		if _, has := seen[cAsset.BaseDenom]; has {
			return fmt.Errorf("duplicate cAsset base denom")
		}

		seen[cAsset.Name] = struct{}{}
		seen[cAsset.BaseDenom] = struct{}{}
	}

	return nil
}

func validateCAsset(dexDenoms []*DexDenom, cAsset *CAsset) error {
	if !hasDenom(dexDenoms, cAsset.BaseDenom) {
		return fmt.Errorf("cAsset's base denom (%v) not found in dex denoms", cAsset.BaseDenom)
	}

	if !hasDenom(dexDenoms, cAsset.Name) {
		return fmt.Errorf("cAsset's denom not found in dex denoms")
	}

	if cAsset.DexFeeShare.IsNil() {
		cAsset.DexFeeShare = math.LegacyZeroDec()
	}

	if cAsset.DexFeeShare.LT(math.LegacyZeroDec()) {
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

func validateCollateralDenoms(v any, dexDenoms []*DexDenom) error {
	collateralDenoms, ok := v.([]*CollateralDenom)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	seen := make(map[string]struct{})

	for _, collateralDenom := range collateralDenoms {
		if err := validateCollateralDenom(dexDenoms, collateralDenom); err != nil {
			return errors.Wrap(err, fmt.Sprintf("error validating collateral denom %v", collateralDenom.Denom))
		}

		if _, has := seen[collateralDenom.Denom]; has {
			return fmt.Errorf("duplicate collateral denom")
		}
		seen[collateralDenom.Denom] = struct{}{}
	}

	return nil
}

func validateCollateralDenom(dexDenoms []*DexDenom, collateralDenom *CollateralDenom) error {
	if collateralDenom.Ltv.IsNil() {
		return fmt.Errorf("ltv is nil")
	}

	if collateralDenom.MaxDeposit.IsNil() {
		return fmt.Errorf("max_deposit is nil")
	}

	if collateralDenom.Ltv.LT(math.LegacyZeroDec()) {
		return fmt.Errorf("ltv must not be smaller than 0")
	}

	if collateralDenom.Ltv.GT(math.LegacyOneDec()) {
		return fmt.Errorf("ltv must not be larger than 1")
	}

	if collateralDenom.MaxDeposit.LT(math.ZeroInt()) {
		return fmt.Errorf("max deposit must not be smaller than 0")
	}

	if !hasDenom(dexDenoms, collateralDenom.Denom) {
		return fmt.Errorf("collateral denom has to be dex denom")
	}

	return nil
}

func validateDexDenoms(v any) error {
	dexDenoms, ok := v.([]*DexDenom)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	seen := make(map[string]struct{})

	for index, dexDenom := range dexDenoms {
		if dexDenom == nil {
			return fmt.Errorf("dex denom #%v is nil", index)
		}

		if err := validateDexDenom(dexDenom); err != nil {
			return errors.Wrap(err, fmt.Sprintf("error validating dex denom %v", dexDenom.Name))
		}

		if _, has := seen[dexDenom.Name]; has {
			return fmt.Errorf("duplicate dex denom: %v", dexDenom.Name)
		}
		seen[dexDenom.Name] = struct{}{}
	}

	return nil
}

func validateDexDenom(dexDenom *DexDenom) error {
	if dexDenom.Name == "" {
		return fmt.Errorf("dex denom name cannot be empty")
	}

	if dexDenom.MinOrderSize.IsNil() {
		return fmt.Errorf("min order size is nil")
	}

	if dexDenom.MinOrderSize.LTE(math.ZeroInt()) {
		return fmt.Errorf("minimum order size has to be bigger than zero")
	}

	if dexDenom.Name != utils.BaseCurrency {
		if dexDenom.Factor == nil || dexDenom.Factor.IsNil() {
			return fmt.Errorf("for dex denoms other than base, factor cannot be nil")
		}

		if !dexDenom.Factor.GT(math.LegacyZeroDec()) {
			return fmt.Errorf("factor must be larger than zero")
		}

		if dexDenom.MinLiquidity.IsNil() {
			return fmt.Errorf("min liquidity is nil")
		}

		if dexDenom.MinLiquidity.LTE(math.ZeroInt()) {
			return fmt.Errorf("minimum liquidty must not be smaller than zero")
		}
	}

	return nil
}

func hasDenom(dexDenoms []*DexDenom, denom string) bool {
	for _, dexDenom := range dexDenoms {
		if dexDenom.Name == denom {
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
