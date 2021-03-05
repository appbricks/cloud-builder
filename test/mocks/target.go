package mocks

import (
	"fmt"

	"github.com/appbricks/cloud-builder/cookbook"
	"github.com/appbricks/cloud-builder/target"
	"github.com/mevansam/gocloud/backend"
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goutils/run"

	backend_mocks "github.com/mevansam/gocloud/test/mocks"
	provider_mocks "github.com/mevansam/gocloud/test/mocks"
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
		"", "", "", ""); err != nil {
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

func NewMockTarget(cli run.CLI) *target.Target {

	return &target.Target{
		RecipeName: "fakeRecipe",
		RecipeIaas: "fakeIAAS",

		Recipe: NewFakeRecipe(cli),
		Provider: provider_mocks.NewFakeCloudProvider(),
		Backend: backend_mocks.NewFakeCloudBackend(),
	}
}
