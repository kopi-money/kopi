package keeper

import (
	"context"

	"github.com/kopi-money/kopi/x/tokenfactory/types"
)

func (k Keeper) QueryCategories(ctx context.Context, _ *types.QueryCategoriesRequest) (*types.QueryCategoriesResponse, error) {
	var categories []types.CategoryResponse

	for _, category := range k.GetParams(ctx).Categories.Categories {
		categories = append(categories, types.CategoryResponse{
			Index:         category.Index,
			Name:          category.Name,
			CreationPrice: category.CreationPrice.String(),
		})
	}

	return &types.QueryCategoriesResponse{Categories: categories}, nil
}
