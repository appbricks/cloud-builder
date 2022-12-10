package config

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/appbricks/cloud-builder/cookbook"
	"github.com/appbricks/cloud-builder/target"
	"github.com/mevansam/gocloud/backend"
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/config"
	"github.com/mevansam/goutils/logger"
)

// global configuration context
type targetContext struct {
	cookbook *cookbook.Cookbook
	targets  *target.TargetSet

	providers map[string]provider.CloudProvider
	backends  map[string]backend.CloudBackend

	dirty bool
}

// in: cookbook - the cookbook in context
func NewConfigContext(cookbook *cookbook.Cookbook) (TargetContext, error) {

	var (
		err error
	)

	ctx := &targetContext{
		cookbook: cookbook,
		dirty:    false,
	}

	if ctx.providers, err = provider.NewCloudProviderTemplates(); err != nil {
		return nil, err
	}
	if ctx.backends, err = backend.NewCloudBackendTemplates(); err != nil {
		return nil, err
	}
	ctx.targets = target.NewTargetSet(ctx)
	return ctx, nil
}

func (cc *targetContext) Reset() error {

	var (
		err error
	)

	if cc.providers, err = provider.NewCloudProviderTemplates(); err != nil {
		return err
	}
	if cc.backends, err = backend.NewCloudBackendTemplates(); err != nil {
		return err
	}
	cc.targets = target.NewTargetSet(cc)
	return nil
}

// loads the cloud configuration from the given stream
func (cc *targetContext) Load(input io.Reader) error {

	type elemType int

	const (
		root elemType = iota
		cloud
		providers
		backends
	)
	const endObject = json.Delim('}')
	elemStack := []elemType{root}

	var (
		err    error
		exists bool
		top    int
		token  json.Token

		cloudProvider provider.CloudProvider
		cloudBackend  backend.CloudBackend
	)

	decoder := json.NewDecoder(input)
	for {
		token, err = decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if decoder.More() {
			top = len(elemStack) - 1

			switch key := token.(type) {
			case json.Delim:
				if key == endObject {
					elemStack = elemStack[0:top]
				}

			case string:

				switch elemStack[top] {
				case root:
					switch key {
					case "cloud":
						elemStack = append(elemStack, cloud)
					default:
						return fmt.Errorf(
							"invalid root config key '%s'",
							key)
					}

				case cloud:
					switch key {
					case "providers":
						elemStack = append(elemStack, providers)

					case "backends":
						elemStack = append(elemStack, backends)

					case "recipes":
						if err = decoder.Decode(cc.cookbook); err != nil {
							return err
						}

					case "targets":
						if err = decoder.Decode(cc.targets); err != nil {
							return err
						}

					default:
						return fmt.Errorf(
							"invalid 'cloud' config key '%s': elemStack = %# v",
							key, elemStack)
					}

				case providers:
					if cloudProvider, exists = cc.providers[key]; !exists {
						return fmt.Errorf(
							"invalid cloud provider '%s'",
							key)
					}
					if err = decoder.Decode(cloudProvider); err != nil {
						return err
					}

				case backends:
					if cloudBackend, exists = cc.backends[key]; !exists {
						return fmt.Errorf(
							"invalid cloud backend '%s'",
							key)
					}
					if err = decoder.Decode(cloudBackend); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// saves the cloud configuration to the given stream
func (cc *targetContext) Save(output io.Writer) error {

	var (
		err error
		i   int
	)
	encoder := json.NewEncoder(output)

	// begin root
	if _, err = output.Write([]byte{'{'}); err != nil {
		return err
	}

	// begin cloud config object
	if _, err = fmt.Fprint(output, "\"cloud\":{"); err != nil {
		return err
	}

	// begin providers
	if _, err = fmt.Fprint(output, "\"providers\":{"); err != nil {
		return err
	}
	i = 0
	for _, p := range cc.providers {
		if i > 0 {
			if _, err = output.Write([]byte{','}); err != nil {
				return err
			}
		}
		if _, err = fmt.Fprintf(output, "\"%s\":", p.Name()); err != nil {
			return err
		}
		if err := encoder.Encode(p); err != nil {
			return err
		}
		i++
	}
	// end providers
	if _, err = output.Write([]byte{'}'}); err != nil {
		return err
	}

	// begin backends
	if _, err = fmt.Fprint(output, ",\"backends\":{"); err != nil {
		return err
	}
	i = 0
	for _, b := range cc.backends {
		if i > 0 {
			if _, err = output.Write([]byte{','}); err != nil {
				return err
			}
		}
		if _, err = fmt.Fprintf(output, "\"%s\":", b.Name()); err != nil {
			return err
		}
		if err := encoder.Encode(b); err != nil {
			return err
		}
		i++
	}
	// end backends
	if _, err = output.Write([]byte{'}'}); err != nil {
		return err
	}

	// encode coookbook
	if _, err = fmt.Fprint(output, ",\"recipes\":"); err != nil {
		return err
	}
	if err = encoder.Encode(cc.cookbook); err != nil {
		return err
	}

	// begin targets
	if _, err = fmt.Fprint(output, ",\"targets\":"); err != nil {
		return err
	}
	if err = encoder.Encode(cc.targets); err != nil {
		return err
	}

	if _, err = output.Write([]byte{
		// end cloud
		'}',
	}); err != nil {
		return err
	}

	if _, err = output.Write([]byte{
		// end root
		'}',
	}); err != nil {
		return err
	}

	return nil
}

func (cc *targetContext) Cookbook() *cookbook.Cookbook {
	return cc.cookbook
}

func (cc *targetContext) GetCookbookRecipe(recipe, iaas string) (cookbook.Recipe, error) {

	var (
		err error

		r    cookbook.Recipe
		copy config.Configurable
	)

	if r = cc.Cookbook().GetRecipe(recipe, iaas); r == nil {
		return nil, fmt.Errorf(
			"recipe '%s' for iaas '%s' does not exist",
			recipe, iaas)
	}
	if copy, err = r.Copy(); err != nil {
		return nil, err
	}
	return copy.(cookbook.Recipe), nil
}

func (cc *targetContext) SaveCookbookRecipe(recipe cookbook.Recipe) {
	cc.cookbook.SetRecipe(recipe)
}

func (cc *targetContext) CloudProviderTemplates() []provider.CloudProvider {

	providerList := []provider.CloudProvider{}
	for _, cp := range cc.providers {
		providerList = append(providerList, cp)
	}

	provider.SortCloudProviders(providerList)
	return providerList
}

func (cc *targetContext) GetCloudProvider(iaas string) (provider.CloudProvider, error) {

	var (
		err error
		ok  bool

		p    provider.CloudProvider
		copy config.Configurable
	)

	if p, ok = cc.providers[iaas]; !ok {
		return nil, fmt.Errorf(
			"provider for iaas '%s' does not exist",
			iaas)
	}
	if copy, err = p.Copy(); err != nil {
		return nil, err
	}
	return copy.(provider.CloudProvider), nil
}

func (cc *targetContext) SaveCloudProvider(provider provider.CloudProvider) {
	cc.providers[provider.Name()] = provider
	cc.dirty = true
}

func (cc *targetContext) GetCloudBackend(name string) (backend.CloudBackend, error) {

	var (
		err error
		ok  bool

		b    backend.CloudBackend
		copy config.Configurable
	)

	if b, ok = cc.backends[name]; !ok {
		return nil, fmt.Errorf(
			"backend of type '%s' does not exist",
			name)
	}
	if copy, err = b.Copy(); err != nil {
		return nil, err
	}
	return copy.(backend.CloudBackend), nil
}

func (cc *targetContext) SaveCloudBackend(backend backend.CloudBackend) {
	cc.backends[backend.Name()] = backend
	cc.dirty = true
}

func (cc *targetContext) NewTarget(
	recipeKey, recipeIaas string,
) (*target.Target, error) {

	var (
		err error

		recipeCopy   cookbook.Recipe
		providerCopy provider.CloudProvider
		backendCopy  backend.CloudBackend
	)

	if recipeCopy, err = cc.GetCookbookRecipe(recipeKey, recipeIaas); err != nil {
		return nil, err
	}
	if providerCopy, err = cc.GetCloudProvider(recipeIaas); err != nil {
		return nil, err
	}
	backendType := recipeCopy.BackendType()
	if len(backendType) != 0 {
		if backendCopy, err = cc.GetCloudBackend(backendType); err != nil {
			return nil, err
		}
	}

	return target.NewTarget(
		recipeCopy,
		providerCopy,
		backendCopy,
	), nil
}

func (cc *targetContext) TargetSet() *target.TargetSet {
	return cc.targets
}

func (cc *targetContext) HasTarget(name string) bool {
	tgt := cc.targets.GetTarget(name)
	return tgt != nil
}

func (cc *targetContext) GetTarget(name string) (*target.Target, error) {

	var (
		tgt *target.Target
	)

	if tgt = cc.targets.GetTarget(name); tgt == nil {
		return nil, fmt.Errorf("target '%s' does not exist", name)
	}
	return tgt.Copy()
}

func (cc *targetContext) SaveTarget(key string, target *target.Target) {
	if err := cc.targets.SaveTarget(key, target); err != nil {
		logger.DebugMessage("Error saving target '%s': %s", key, err.Error())
	} else {
		cc.dirty = true
	}
}

func (cc *targetContext) DeleteTarget(key string) {
	cc.targets.DeleteTarget(key)
	cc.dirty = true
}

func (cc *targetContext) IsDirty() bool {
	return cc.dirty
}