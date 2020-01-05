package mocks

import (
	"fmt"

	"github.com/mevansam/gocloud/backend"
	"github.com/mevansam/gocloud/provider"
	"github.com/appbricks/cloud-builder/cookbook"
	"github.com/appbricks/cloud-builder/target"
)

type FakeTargetContext struct {
	recipePath string
}

func NewTargetMockContext(
	recipePath string,
) *FakeTargetContext {

	return &FakeTargetContext{
		recipePath: recipePath,
	}
}

func (mctx *FakeTargetContext) NewTarget(
	recipeName, recipeIaas string,
) (*target.Target, error) {

	var (
		err error

		r cookbook.Recipe
		p provider.CloudProvider
		b backend.CloudBackend
	)

	if r, err = cookbook.NewRecipe(
		recipeName,
		recipeIaas,
		fmt.Sprintf("%s/%s/%s", mctx.recipePath, recipeName, recipeIaas),
		"", "", ""); err != nil {
		return nil, err
	}
	if p, err = provider.NewCloudProvider(recipeIaas); err != nil {
		return nil, err
	}
	if b, err = backend.NewCloudBackend(r.BackendType()); err != nil {
		return nil, err
	}

	return target.NewTarget(r, p, b), nil
}
