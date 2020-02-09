package cookbook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/gobuffalo/packr/v2"
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/utils"
)

const (
	cookbookZipFile = "cookbook.zip"
	cookbookModTime = "cookbook-mod-time"
)

type Cookbook struct {
	path  string
	files []string

	// nested map [recipe_name][iaas_name]
	recipes map[string]map[string]Recipe
}

type CookbookRecipeInfo struct {
	Name     string
	IaaSList []provider.CloudProvider
}

var recipePathMatcher = regexp.MustCompile(
	fmt.Sprintf("^recipes\\%c.*\\%c\\.terraform\\%c?$",
		os.PathSeparator, os.PathSeparator, os.PathSeparator,
	),
)
var filePathSeparator = fmt.Sprintf("%c", os.PathSeparator)

func NewCookbook(
	box *packr.Box,
	workspacePath string,
	outputBuffer, errorBuffer io.Writer,
) (*Cookbook, error) {

	var (
		err error
		ok  bool

		cookbookTimestamp,
		tfPluginPath,
		tfCLIPath string

		c *Cookbook
		r Recipe
	)

	if cookbookTimestamp, err = box.FindString(cookbookModTime); err != nil {
		return nil, err
	}
	cookbookTimestamp = strings.Trim(cookbookTimestamp, "\n")

	c = &Cookbook{
		path:    filepath.Join(workspacePath, "cookbook", cookbookTimestamp),
		recipes: make(map[string]map[string]Recipe),
	}

	// Updates cookbook metadata
	addMetadata := func(file string) error {

		var (
			match bool
			rr    map[string]Recipe
		)

		if match = recipePathMatcher.Match([]byte(file)); match {

			elems := strings.Split(file, filePathSeparator)
			if len(elems) >= 4 {
				pathSuffix := filepath.Join(elems[0], elems[1], elems[2])

				name := elems[1]
				iaas := elems[2]

				if rr, ok = c.recipes[name]; !ok {
					rr = make(map[string]Recipe)
					c.recipes[name] = rr
				}
				if r, err = NewRecipe(
					name,
					iaas,
					filepath.Join(c.path, pathSuffix),
					tfPluginPath,
					tfCLIPath,
					filepath.Join(workspacePath, "run", pathSuffix),
					cookbookTimestamp,
				); err != nil {
					return err
				}

				logger.TraceMessage("Initialized recipe: %#v\n", r)
				rr[iaas] = r
			}
		}
		return nil
	}

	info, err := os.Stat(c.path)
	if os.IsNotExist(err) {

		cookbookZip, err := box.FindString(cookbookZipFile)
		if err != nil {
			return nil, err
		}
		if _, err = utils.Unzip([]byte(cookbookZip), c.path); err != nil {
			return nil, err
		}
		info, _ = os.Stat(c.path)
		logger.TraceMessage("Unzipped cookbook to %s:\n  %s\n\n", c.path, strings.Join(c.files, "\n  "))
	}
	if info.IsDir() {

		// embedded cookbook plugin path
		tfPluginPath = filepath.Join(
			c.path, "bin", "plugins",
			fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH),
		)
		if runtime.GOOS == "windows" {
			// windows cli
			tfCLIPath = filepath.Join(c.path, "bin", "terraform.exe")
		} else {
			// *nix cli
			tfCLIPath = filepath.Join(c.path, "bin", "terraform")
		}

		// Retrieve cookbook file list by walking
		// the extracted cookbook's directory tree
		c.files = []string{}

		pathPrefixLen := len(c.path) + 1
		err = filepath.Walk(c.path,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if len(path) >= pathPrefixLen {
					filepath := path[pathPrefixLen:]
					c.files = append(c.files, filepath)

					err = addMetadata(filepath)
					if err != nil {
						return err
					}
				}
				return nil
			})

		if err != nil {
			return nil, err
		}
		logger.TraceMessage("Initialized cookbook: %# v", c)

	} else {
		return nil, fmt.Errorf("cookbook path '%s' exists but is not a directory", c.path)
	}
	return c, nil
}

func (c *Cookbook) Validate() error {

	var (
		err error
	)

	// Validate files in cookbook
	for _, f := range c.files {

		if _, err = os.Stat(filepath.Join(c.path, f)); !os.IsNotExist(err) {
			return err
		}
	}

	// Validate cookbook recipes
	if len(c.recipes) == 0 {
		return fmt.Errorf("no recipes in cookbook")
	}

	for name, rr := range c.recipes {

		if len(rr) == 0 {
			return fmt.Errorf(
				"recipe '%s' in cookbook has no IaaS specific templates",
				name,
			)
		}

		for iaas, r := range rr {
			if _, err = os.Stat(
				filepath.Join(
					c.path,
					"..",
					"run",
					"recipes",
					name,
					iaas,
					".terraform",
				),
			); !os.IsNotExist(err) {
				return err
			}
			if err = r.(*recipe).validate(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Cookbook) IaaSList() []provider.CloudProvider {

	iaasSet := make(map[string]provider.CloudProvider)
	for _, rr := range c.recipes {
		for iaas := range rr {
			cp, _ := provider.NewCloudProvider(iaas)
			iaasSet[iaas] = cp
		}
	}
	iaasList := []provider.CloudProvider{}
	for _, cp := range iaasSet {
		iaasList = append(iaasList, cp)
	}

	provider.SortCloudProviders(iaasList)
	return iaasList
}

func (c *Cookbook) RecipeList() []CookbookRecipeInfo {

	recipeInfos := make([]CookbookRecipeInfo, 0, len(c.recipes))
	l := 0

	for name, rr := range c.recipes {
		recipeInfo := CookbookRecipeInfo{
			Name:     name,
			IaaSList: []provider.CloudProvider{},
		}

		i := sort.Search(l, func(j int) bool {
			return recipeInfos[j].Name > name
		})
		recipeInfos = append(recipeInfos, recipeInfo)
		if len(recipeInfos) > 1 {
			copy(recipeInfos[i+1:], recipeInfos[i:])
			recipeInfos[i] = recipeInfo
		}
		l++

		for iaas := range rr {
			cp, _ := provider.NewCloudProvider(iaas)
			recipeInfos[i].IaaSList = append(recipeInfos[i].IaaSList, cp)
		}
		provider.SortCloudProviders(recipeInfos[i].IaaSList)
	}
	return recipeInfos
}

func (c *Cookbook) HasRecipe(recipe, iaas string) bool {

	var (
		ok bool
		rr map[string]Recipe
	)

	if rr, ok = c.recipes[recipe]; ok {
		_, ok = rr[iaas]
	}
	return ok
}

func (c *Cookbook) GetRecipe(recipe, iaas string) Recipe {

	var (
		ok bool
		rr map[string]Recipe
		r  Recipe
	)

	r = nil
	if rr, ok = c.recipes[recipe]; ok {
		r = rr[iaas]
	}
	return r
}

func (c *Cookbook) SetRecipe(recipe Recipe) {

	var (
		ok bool
		rr map[string]Recipe
	)

	nameElements := strings.Split(recipe.Name(), "/")

	if rr, ok = c.recipes[nameElements[0]]; !ok {
		rr = make(map[string]Recipe)
		c.recipes[nameElements[0]] = rr
	}
	rr[nameElements[1]] = recipe
}

// interface: encoding/json/Unmarshaler

func (c *Cookbook) UnmarshalJSON(b []byte) error {

	type keyType int

	const (
		none keyType = iota
		recipe
		name
	)

	var (
		err   error
		token json.Token

		currKey    keyType
		recipeName string
	)
	decoder := json.NewDecoder(bytes.NewReader(b))

	// read array open bracket
	if _, err = utils.ReadJSONDelimiter(decoder, utils.JsonArrayStartDelim); err != nil {
		return err
	}

	// read recipe configurations
	currKey = none
	for {
		if token, err = decoder.Token(); token == utils.JsonArrayEndDelim || err != nil {
			break
		}

		switch t := token.(type) {

		case json.Delim:
			switch t {
			case utils.JsonObjectStartDelim:
				currKey = recipe
			case utils.JsonObjectEndDelim:
				recipeName = ""
				currKey = none
			default:
				return fmt.Errorf(
					"unexpected json delimiter encountered while parsing IaaS configuration document")
			}

		case string:

			switch currKey {
			case recipe:
				switch t {
				case "name":
					if len(recipeName) > 0 {
						return fmt.Errorf(
							"config recipe '%s' is still being parsed when another name was encountered",
							recipeName)
					}
					currKey = name
				case "config":
					if err = c.readRecipeIaaSConfigs(recipeName, decoder); err != nil {
						return err
					}
					currKey = recipe

				default:
					return fmt.Errorf("unknown recipe config key '%s'", t)
				}

			case name:
				recipeName = t
				currKey = recipe
			}
		}
	}
	if err == io.EOF {
		return nil
	}
	return err
}

func (c *Cookbook) readRecipeIaaSConfigs(recipeName string, decoder *json.Decoder) error {

	var (
		err   error
		token json.Token
		r     Recipe
	)

	// read start of iaas specific recipe configurations
	if _, err = utils.ReadJSONDelimiter(decoder, utils.JsonObjectStartDelim); err != nil {
		return err
	}
	for {
		if token, err = decoder.Token(); token == utils.JsonObjectEndDelim || err != nil {
			break
		}

		switch t := token.(type) {

		case string:
			if r = c.GetRecipe(recipeName, t); r == nil {
				return fmt.Errorf(
					"recipe '%s' for IaaS '%s' was not found",
					recipeName, t)
			}
			if err = decoder.Decode(r); err != nil {
				return err
			}
		}
	}

	return err
}

// interface: encoding/json/Marshaler

func (c *Cookbook) MarshalJSON() ([]byte, error) {

	var (
		err            error
		out            bytes.Buffer
		first1, first2 bool
	)
	encoder := json.NewEncoder(&out)

	out.WriteRune('[')
	first1 = true

	for name, rr := range c.recipes {
		if first1 {
			first1 = false
		} else {
			out.WriteRune(',')
		}

		out.WriteString("{\"name\":\"")
		out.WriteString(name)
		out.WriteString("\",\"config\":{")
		first2 = true

		for iaas, r := range rr {
			if first2 {
				first2 = false
			} else {
				out.WriteRune(',')
			}

			out.WriteRune('"')
			out.WriteString(iaas)
			out.WriteString("\":")
			if err = encoder.Encode(r); err != nil {
				return nil, err
			}
		}

		out.WriteString("}}")
	}

	out.WriteRune(']')
	return out.Bytes(), nil
}
