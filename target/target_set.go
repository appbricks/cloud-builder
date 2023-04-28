package target

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/appbricks/cloud-builder/cookbook"
	"github.com/appbricks/cloud-builder/terraform"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/utils"
)

type TargetSet struct {
	ctx context

	targets map[string]*Target
	disabledTargets []*parsedTarget
}

// temporary target data structure used
// when parsing serialized targets in
// order to resolve the configurable types
type parsedTarget struct {
	RecipeName string `json:"recipeName"`
	RecipeIaas string `json:"recipeIaas"`

	CookbookName    string `json:"cookbookName"`
	CookbookVersion string `json:"cookbookVersion"`
	RepoTimestamp   string `json:"repoTimestamp,omitempty"`

	DependentTargets []string `json:"dependentTargets"`

	Recipe   json.RawMessage `json:"recipe"`
	Provider json.RawMessage `json:"provider"`
	Backend  json.RawMessage `json:"backend"`

	Output *map[string]terraform.Output `json:"output,omitempty"`

	RSAPrivateKey string `json:"rsaPrivateKey,omitempty"`
	RSAPublicKey  string `json:"rsaPublicKey,omitempty"`

	NodeKey string `json:"nodeKey,omitempty"`
	NodeID  string `json:"nodeID,omitempty"`
}

// interface definition of global config context
// specific to TargetSet. declared here to simplify
// mocking and avoid cyclical dependencies.
type context interface {
	NewTarget(
		recipeKey,
		recipeIaas string,
	) (*Target, error)
}

func NewTargetSet(ctx context) *TargetSet {

	return &TargetSet{
		ctx:     ctx,
		targets: make(map[string]*Target),
	}
}

func (ts *TargetSet) Lookup(keyValues ...string) []*Target {

	keyPath := CreateKey(keyValues...)

	targets := make([]*Target, 0, len(ts.targets))
	l := 0

	for _, t := range ts.targets {

		if strings.HasPrefix(t.Key(), keyPath) {
			// add targets to array
			// sorting it along the way
			i := sort.Search(l, func(j int) bool {
				return targets[j].Key() > t.Key()
			})
			targets = append(targets, nil)
			if targets[i] != nil {
				copy(targets[i+1:], targets[i:])
			}
			targets[i] = t
			l++
		}
	}
	return targets
}

func (ts *TargetSet) GetTargets() []*Target {

	targets := make([]*Target, 0, len(ts.targets))
	l := 0

	for _, t := range ts.targets {
		// add targets to array
		// sorting it along the way
		i := sort.Search(l, func(j int) bool {
			return targets[j].Key() > t.Key()
		})
		targets = append(targets, nil)
		if targets[i] != nil {
			copy(targets[i+1:], targets[i:])
		}
		targets[i] = t
		l++
	}
	return targets
}

func (ts *TargetSet) GetTarget(name string) *Target {
	logger.TraceMessage(
		"Retrieving target with name '%s' from: %+v",
		name, ts.targets)

	return ts.targets[name]
}

func (ts *TargetSet) SaveTarget(key string, target *Target) error {
	logger.TraceMessage("Saving target: %# v", target)

	target.dependencies = []*Target{}
	if len(target.DependentTargets) > 0 {
		for _, dependentTarget := range target.DependentTargets {
			t := ts.targets[dependentTarget]
			if t == nil {
				return fmt.Errorf(
					"Dependent target '%s' of target '%s' was not found", 
					dependentTarget, key,
				)
			}
			target.dependencies = append(target.dependencies, t)
			t.dependents++
		}
	}

	// delete target with given key before
	// saving in the target map, as the key of
	// the new/updated target may have changed
	delete(ts.targets, key)
	ts.targets[target.Key()] = target

	return nil
}

func (ts *TargetSet) DeleteTarget(key string) {
	logger.TraceMessage("Deleting target with key. %s", key)
	delete(ts.targets, key)
}

func (ts *TargetSet) GetDisabledTargetRecipes() []cookbook.CookbookRecipeInfo {

	recipesAdded := make(map[string]bool)
	recipeInfos := make([]cookbook.CookbookRecipeInfo, 0, len(ts.disabledTargets))	
	l := 0

	for _, target := range ts.disabledTargets {
		recipeInfo := cookbook.CookbookRecipeInfo{
			CookbookName: target.CookbookName,
			CookbookVersion: target.CookbookVersion,
			RecipeName: target.RecipeName,
		}
		key := strings.Join([]string{ target.CookbookName, target.CookbookVersion, target.RecipeName }, "/")
		if recipesAdded[key] {
			continue
		}
		recipesAdded[key] = true

		// insert recipe in descending order of cookebook and recipe name
		// (bastion recipes are added to the head of the list)
		i := sort.Search(l, func(j int) bool {
			return recipeInfo.CookbookName < recipeInfos[j].CookbookName ||
				recipeInfo.CookbookVersion < recipeInfos[j].CookbookVersion ||
				recipeInfo.RecipeName < recipeInfos[j].RecipeName
		})
		recipeInfos = recipeInfos[:l+1]
		if i == l {
			recipeInfos[l] = recipeInfo
		} else {
			copy(recipeInfos[i+1:], recipeInfos[i:])
			recipeInfos[i] = recipeInfo
		}
		l++
	}	
	return recipeInfos
}

// interface: encoding/json/Unmarshaler

func (ts *TargetSet) UnmarshalJSON(b []byte) error {

	var (
		err error

		target *Target
	)

	decoder := json.NewDecoder(bytes.NewReader(b))

	// read array open bracket
	if _, err = utils.ReadJSONDelimiter(decoder, utils.JsonArrayStartDelim); err != nil {
		return err
	}

	targetsWithDependencies := []*Target{}

	for decoder.More() {

		parsedTarget := parsedTarget{}
		if err = decoder.Decode(&parsedTarget); err != nil {
			return err
		}

		if target, err = ts.ctx.NewTarget(
			parsedTarget.CookbookName + ":" + parsedTarget.RecipeName,
			parsedTarget.RecipeIaas,
		); err != nil {
			ts.disabledTargets = append(ts.disabledTargets, &parsedTarget)
			logger.ErrorMessage("Unable to load saved target: %s", err.Error())
			continue
		}
		if err = json.Unmarshal(parsedTarget.Recipe, target.Recipe); err != nil {
			return err
		}
		if parsedTarget.Provider != nil && target.Provider != nil {
			if err = json.Unmarshal(parsedTarget.Provider, target.Provider); err != nil {
				return err
			}
		}
		if parsedTarget.Backend != nil && target.Backend != nil {
			if err = json.Unmarshal(parsedTarget.Backend, target.Backend); err != nil {
				return err
			}	
		}
		target.DependentTargets = parsedTarget.DependentTargets
		target.Output = parsedTarget.Output
		target.CookbookName = parsedTarget.CookbookName
		target.CookbookVersion = parsedTarget.CookbookVersion
		target.RepoTimestamp = parsedTarget.RepoTimestamp
		target.RSAPrivateKey = parsedTarget.RSAPrivateKey
		target.RSAPublicKey = parsedTarget.RSAPublicKey
		target.NodeKey = parsedTarget.NodeKey
		target.NodeID = parsedTarget.NodeID
		target.Refresh()

		if len(target.DependentTargets) > 0 {
			targetsWithDependencies = append(targetsWithDependencies, target)
		} else {
			ts.targets[target.Key()] = target
		}
	}

	OUTER:
	for _, target := range targetsWithDependencies {
		for _, dependentTarget := range target.DependentTargets {
			t := ts.targets[dependentTarget]
			if t == nil {
				logger.DebugMessage(
					"Dependent target '%s' of target '%s' was not found. Target will be deleted.", 
					dependentTarget, target.Key())

				delete(ts.targets, target.Key())
				continue OUTER
			}
			target.dependencies = append(target.dependencies, t)
			t.dependents++
		}
		ts.targets[target.Key()] = target
	}

	// read array close bracket
	if _, err = utils.ReadJSONDelimiter(decoder, utils.JsonArrayEndDelim); err != nil {
		return err
	}

	return nil
}

// interface: encoding/json/Marshaler

func (ts *TargetSet) MarshalJSON() ([]byte, error) {

	var (
		err error
		out bytes.Buffer
	)
	encoder := json.NewEncoder(&out)

	if _, err = out.WriteRune('['); err != nil {
		return out.Bytes(), err
	}

	first := true
	for _, target := range ts.targets {
		if first {
			first = false
		} else {
			out.WriteRune(',')
		}

		if err = encoder.Encode(target); err != nil {
			return out.Bytes(), err
		}
	}
	for _, target := range ts.disabledTargets {
		if first {
			first = false
		} else {
			out.WriteRune(',')
		}

		if err = encoder.Encode(target); err != nil {
			return out.Bytes(), err
		}
	}

	if _, err = out.WriteRune(']'); err != nil {
		return out.Bytes(), err
	}

	return out.Bytes(), nil
}
