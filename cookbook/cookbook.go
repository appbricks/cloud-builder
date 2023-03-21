package cookbook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/gobuffalo/packr/v2"
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/utils"
	"gopkg.in/yaml.v2"
)

const (
	cookbookZipFile = "cookbook.zip"
	cookbookModTime = "cookbook-mod-time"
)

type Cookbook struct {
	workspacePath string

	path,
	tfPluginPath,
	tfCLIPath string

	files []string
	
	// nested map [recipe_name][iaas_name]
	recipes map[string]map[string]Recipe

	cookbooks     map[string]*CookbookMetadata
	repoTimestamp string

	mx   sync.Mutex
	init sync.WaitGroup
	errs []error
}

type CookbookRecipeInfo struct {
	RecipeKey string
	
	CookbookName    string
	CookbookVersion string
	RecipeName      string

	IsBastion bool
	IaaSList  []provider.CloudProvider
}

type CookbookMetadata struct {
	CookbookName     string `yaml:"cookbook-name"`
	CookbookVersion  string `yaml:"cookbook-version"`
	Description      string `yaml:"description"`
	TerraformVersion string `yaml:"terraform-version"`
	TargetOsName     string `yaml:"target-os-name"`
	TargetOsArch     string `yaml:"target-os-arch"`

	EnvVars [][]string `yaml:"env-args"`

	Imported bool
	Recipes  []string

	cookbookPath string
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

		ts string
		c  *Cookbook

		importedCookbooks []os.DirEntry
		vbytes            []byte
	)
	newCoreCookbook := false

	if ts, err = box.FindString(cookbookModTime); err != nil {
		return nil, err
	}
	ts = strings.Trim(ts, "\n")

	c = &Cookbook{
		workspacePath: workspacePath,

		path:    filepath.Join(workspacePath, "cookbook", ts),
		recipes: make(map[string]map[string]Recipe),

		cookbooks:     make(map[string]*CookbookMetadata),
		repoTimestamp: ts,
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

		newCoreCookbook = true
	}
	if info.IsDir() {

		// embedded cookbook plugin path
		c.tfPluginPath = filepath.Join(c.path, "bin", "plugins")
		if runtime.GOOS == "windows" {
			// windows cli
			c.tfCLIPath = filepath.Join(c.path, "bin", "terraform.exe")
		} else {
			// *nix cli
			c.tfCLIPath = filepath.Join(c.path, "bin", "terraform")
		}

		// Retrieve cookbook file list by walking
		// the extracted cookbook's directory tree
		c.files = []string{}

		// add core recipes
		if err = c.addRecipeMetadata(c.path, filepath.Join(c.path, "recipes")); err != nil {
			return nil, err
		}
		// add recipes from imported cookbooks
		importedPath := filepath.Join(c.workspacePath, "cookbook", "library")
		if importedCookbooks, err = os.ReadDir(importedPath); err == nil {
			for _, ic := range importedCookbooks {
				icpath := filepath.Join(importedPath, ic.Name())
				if newCoreCookbook {
					err = c.importCookbook(icpath)

				} else if vbytes, err = os.ReadFile(filepath.Join(icpath, "CURRENT")); err == nil {
					vpath := filepath.Join(icpath, strings.TrimSpace(string(vbytes[:])))
					err = c.addRecipeMetadata(vpath, filepath.Join(vpath, "recipes"))
				}
				if err != nil {
					return nil, err
				}
			}
		}

		c.init.Wait()
		if len(c.errs) > 0 {
			return nil, c.errs[0]
		}
		logger.DebugMessage("Initialized cookbook at '%s'.", c.path)

	} else {
		return nil, fmt.Errorf("cookbook path '%s' exists but is not a directory", c.path)
	}
	return c, nil
}

func (c *Cookbook) addRecipeMetadata(cookbookRoot, recipesPath string) error {

	var (
		err error
	)

	// Updates cookbook metadata
	addMetadata := func(file string) error {

		var (
			err error
			ok  bool

			recipeKey string

			r  Recipe
			rr map[string]Recipe

			data []byte
			cm   *CookbookMetadata		
		)

		if match := recipePathMatcher.Match([]byte(file)); match {

			elems := strings.Split(file, filePathSeparator)
			if len(elems) >= 4 {
				pathSuffix := filepath.Join(elems[0], elems[1], elems[2])

				recipeName := elems[1]
				recipeIaaS := elems[2]

				metadata := CookbookMetadata{}
				if data, err = os.ReadFile(filepath.Join(cookbookRoot, "METADATA")); err != nil {
					return err
				}
				if err = yaml.Unmarshal(data, &metadata); err != nil {
					return err
				}
				
				recipeKey = metadata.CookbookName + ":" + recipeName

				c.mx.Lock()

				// add/update recipe's cookbook
				if cm, ok = c.cookbooks[metadata.CookbookName]; !ok {
					metadata.Imported = (c.path != cookbookRoot)
					metadata.cookbookPath = cookbookRoot
					cm = &metadata
					c.cookbooks[metadata.CookbookName] = cm
				}
				l := len(cm.Recipes)
				i := sort.Search(l, func(j int) bool {
					return recipeName <= cm.Recipes[j]
				})
				if i == l || recipeName != cm.Recipes[i] {
					cm.Recipes = append(cm.Recipes, recipeName)
					if i < l {
						copy(cm.Recipes[i+1:], cm.Recipes[i:])
						cm.Recipes[i] = recipeName	
					}
				}

				// add update recipe map
				if rr, ok = c.recipes[recipeKey]; !ok {
					rr = make(map[string]Recipe)
					c.recipes[recipeKey] = rr					
				}

				c.mx.Unlock()

				if r, err = NewRecipe(
					recipeKey,
					recipeIaaS,
					filepath.Join(cookbookRoot, pathSuffix),
					c.tfPluginPath,
					filepath.Join(c.workspacePath, "state", cm.CookbookName, pathSuffix),
					c.tfCLIPath,
					filepath.Join(c.workspacePath, "run", cm.CookbookName, pathSuffix),
					c.repoTimestamp,
					metadata.CookbookName,
					metadata.CookbookVersion,
					recipeName,
					metadata.EnvVars,
				); err != nil {
					logger.ErrorMessage("Error loading recipe '%s/%s': %s", recipeKey, recipeIaaS, err.Error())
					return err
				}
				c.mx.Lock()
				rr[recipeIaaS] = r
				c.mx.Unlock()
				
				logger.DebugMessage("Initialized recipe '%s'.\n", r.Name())
				logger.TraceMessage("Recipe metadata for '%s': %# v\n", r.Name(), r)
			}
		}
		return nil
	}

	pathPrefixLen := len(cookbookRoot) + 1
	
	if err = filepath.WalkDir(recipesPath,
		func(path string, de fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if len(path) >= pathPrefixLen {
				filepath := path[pathPrefixLen:]
				c.files = append(c.files, filepath)

				c.init.Add(1)
				go func() {
					defer c.init.Done()						
					
					if err = addMetadata(filepath); err != nil {
						c.mx.Lock()
						c.errs = append(c.errs, err)
						c.mx.Unlock()
					}	
				}()
			}
			return nil
		},
	); err != nil {
		return err
	}

	return nil
}

func (c *Cookbook) ImportCookbook(cookbookPath string) (err error) {

	var (
		ok  bool

		fi      os.FileInfo
		zipFile *os.File

		data  []byte
	)

	libraryPath := filepath.Join(
		c.workspacePath, 
		"cookbook", 
		"library",
	)

	// extract cookbook to be imported
	unzipPath := filepath.Join(libraryPath, ".new")
	os.RemoveAll(unzipPath)
	if err = os.MkdirAll(unzipPath, 0755); err != nil {
		return err
	}
	defer os.RemoveAll(unzipPath)

	if fi, err = os.Stat(cookbookPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("cookbook '%s' file to import does not exist", cookbookPath)
		}
		return err
	}
	if zipFile, err = os.Open(cookbookPath); err != nil {
		return err
	}
	if _, err = utils.UnzipStream(zipFile, fi.Size(), unzipPath); err != nil {
		return err
	}

	// validate cookbook structure
	invalidError := fmt.Errorf("invalid cookbook structure")
	if fi, err = os.Stat(filepath.Join(unzipPath, "bin", "plugins", "registry.terraform.io")); os.IsNotExist(err) || !fi.IsDir() {
		return invalidError
	}
	if fi, err = os.Stat(filepath.Join(unzipPath, "recipes")); os.IsNotExist(err) || !fi.IsDir() {
		return invalidError
	}

	// read cookbook metadata
	metadata := CookbookMetadata{}
	if data, err = os.ReadFile(filepath.Join(unzipPath, "METADATA")); err != nil {
		return err
	}
	if err = yaml.Unmarshal(data, &metadata); err != nil {
		return err
	}
	if len(metadata.CookbookName) == 0 ||
		len(metadata.CookbookVersion) == 0 ||
		len(metadata.TerraformVersion) == 0 ||
		len(metadata.TargetOsName) == 0 ||
		len(metadata.TargetOsArch) == 0 {
		return invalidError
	}
	if runtime.GOOS != metadata.TargetOsName {
		return fmt.Errorf("cookbook does not support local system's os")
	}
	if runtime.GOARCH != metadata.TargetOsArch {
		return fmt.Errorf("cookbook does not support local system os' architecture")
	}

	// import cookbook
	importPath := filepath.Join(
		libraryPath, 
		metadata.CookbookName,
	)
	if err = os.MkdirAll(importPath, 0755); err != nil {
		return err
	}
	currentVersionFile := filepath.Join(importPath, "CURRENT")
	defer func() {
		if err != nil {
			if _, e := os.Stat(currentVersionFile); os.IsNotExist(e) {
				os.RemoveAll(importPath)
			}	
		}
	}()
	
	// move unzipped cookbook to versioned path
	versionedPath := filepath.Join(importPath, metadata.CookbookVersion)
	if _, err = os.Stat(versionedPath); os.IsNotExist(err) {
		if err = os.Rename(unzipPath, versionedPath); err != nil {
			return err
		}
		
	} else if err != nil {
		return err

	} else {
		// if cookbook version to be imported exists then check if the content match
		if ok, err = utils.DirCompare(unzipPath, versionedPath); err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("cookbook version appears to have already been imported but does not match cookook being imported")
		}
	}

	os.Remove(currentVersionFile)
	if err = os.WriteFile(currentVersionFile, []byte(metadata.CookbookVersion), 0644); err != nil {
		return err
	}

	return c.importCookbook(importPath)
}

func (c *Cookbook) importCookbook(cookbookPath string) error {

	var (
		err error

		vbytes []byte
	)

	if vbytes, err = os.ReadFile(filepath.Join(cookbookPath, "CURRENT")); err != nil {
		return err
	}
	versionedPath := filepath.Join(cookbookPath, strings.TrimSpace(string(vbytes[:])))
	pluginSrcPath := filepath.Join(versionedPath, "bin", "plugins", "registry.terraform.io")
	pluginDestPath := filepath.Join(c.path, "bin", "plugins", "registry.terraform.io")

	errs := []error{}

	if err = filepath.WalkDir(pluginSrcPath, func(p string, de fs.DirEntry, err error) error {	

		if err != nil {
			return err
		}
		if de.Type().IsRegular() {

			c.init.Add(1)
			go func() {
				defer c.init.Done()

				destPath := filepath.Join(pluginDestPath, strings.TrimPrefix(p, pluginSrcPath))			
				if _, err = os.Stat(destPath); os.IsNotExist(err) {
					if err = os.MkdirAll(filepath.Dir(destPath), 0755); err == nil {
						if err = utils.CopyFiles(p, destPath, 1024); err != nil {
							os.Remove(destPath)
						}	else {
							_ = os.Chmod(destPath, 0755)
						}
					}
				}
				if err != nil {
					logger.ErrorMessage("Error adding imported cookbook plugin to main: %s", err.Error())
					errs = append(errs, err)
				}
			}()
		}

		return nil
	}); err != nil {
		return err
	}

	// load all recipes in imported cookbook
	if err = c.addRecipeMetadata(versionedPath, filepath.Join(versionedPath, "recipes")); err != nil {
		return err
	}

	c.init.Wait()
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (c *Cookbook) GetCookbook(name string) *CookbookMetadata {
	return c.cookbooks[name]
}

func (c *Cookbook) CookbookList(importedOnly bool) []*CookbookMetadata {

	cookbookList := make([]*CookbookMetadata, 0, len(c.cookbooks))
	l := 0

	// create sorted cookbook list 
	// in order of cookbook name
	for _, cm := range c.cookbooks {
		if importedOnly && !cm.Imported {
			// skip embedded cookooks
			continue
		}

		i := sort.Search(l, func(j int) bool {
			return (!cm.Imported && cookbookList[j].Imported) ||
				(cm.Imported == cookbookList[j].Imported && cm.CookbookName < cookbookList[j].CookbookName)
		})
		cookbookList = cookbookList[:l+1]
		if i == l {
			cookbookList[l] = cm
		} else {
			copy(cookbookList[i+1:], cookbookList[i:])
			cookbookList[i] = cm	
		}
		l++
	}
	return cookbookList
}

func (c *Cookbook) DeleteImportedCookbook(name string) error {

	var (
		err error
	)

	cm := c.cookbooks[name]
	if cm != nil {

		if !cm.Imported {
			return fmt.Errorf("embedded cookbook '%s' cannot be delete", name)
		}

		cookbookLibraryPath := filepath.Dir(cm.cookbookPath)
		if err = os.RemoveAll(cookbookLibraryPath); err != nil {
			return err
		}

		// remove all cookbook recipes
		recipePrefix := name + ":"
		for key := range c.recipes {
			if strings.HasPrefix(key, recipePrefix) {
				delete(c.recipes, key)
			}
		}
	}
	return nil
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

	var (
		iaas string

		key string
		rr  map[string]Recipe
		r   Recipe

		recipeInfo CookbookRecipeInfo
	)

	recipeInfos := make([]CookbookRecipeInfo, 0, len(c.recipes))
	l := 0

	for key, rr = range c.recipes {
		recipeInfo = CookbookRecipeInfo{
			IaaSList: []provider.CloudProvider{},
		}

		// add iaas list
		for iaas, r = range rr {
			cp, _ := provider.NewCloudProvider(iaas)
			recipeInfo.IaaSList = append(recipeInfo.IaaSList, cp)
		}
		provider.SortCloudProviders(recipeInfo.IaaSList)

		recipeInfo.RecipeKey = key
		recipeInfo.CookbookName = r.CookbookName()
		recipeInfo.CookbookVersion = r.CookbookVersion()
		recipeInfo.RecipeName = r.RecipeName()

		recipeInfo.IsBastion = r.IsBastion()

		// insert recipe in descending order of cookebook and recipe name
		// (bastion recipes are added to the head of the list)
		i := sort.Search(l, func(j int) bool {
			return (recipeInfo.IsBastion && !recipeInfos[j].IsBastion) ||
				(recipeInfo.IsBastion == recipeInfos[j].IsBastion &&
					(recipeInfo.CookbookName < recipeInfos[j].CookbookName ||
					recipeInfo.RecipeName < recipeInfos[j].RecipeName))
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

func (c *Cookbook) HasRecipe(recipeKey, iaas string) bool {
	c.init.Wait()

	var (
		ok bool
		rr map[string]Recipe
	)

	if rr, ok = c.recipes[recipeKey]; ok {
		_, ok = rr[iaas]
	}
	return ok
}

func (c *Cookbook) GetRecipe(recipeKey, iaas string) Recipe {
	c.init.Wait()

	var (
		ok bool
		rr map[string]Recipe
		r  Recipe
	)

	r = nil
	if rr, ok = c.recipes[recipeKey]; ok {
		r = rr[iaas]
	}
	return r
}

func (c *Cookbook) SetRecipe(recipe Recipe) {

	var (
		ok bool

		recipeKey string
		rr        map[string]Recipe
	)

	recipeKey = recipe.RecipeKey()
	if rr, ok = c.recipes[recipeKey]; !ok {
		rr = make(map[string]Recipe)
		c.recipes[recipeKey] = rr
	}
	rr[recipe.RecipeIaaS()] = recipe
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
				logger.ErrorMessage(
					"Configured cookbook recipe '%s' for IaaS '%s' was not found",
					recipeName, t,
				)

				// TODO: there appears to be a race condition here where we get
				// a recipe not found error intermittently
				
				// TODO: for now we bind to a recipe instance which we discard.
				// we need to handle this correctly if a target was created
				// with this recipe that no longer exists in the cookbook.
				r = &recipe{
					name: recipeName,
					variables: make(map[string]*Variable),
				}
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
