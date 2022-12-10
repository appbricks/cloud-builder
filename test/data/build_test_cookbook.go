package data

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"

	"github.com/onsi/ginkgo"
)

var (
	testCookbookBuilt bool = false
)

// Build the test cookbook
func EnsureCookbookIsBuilt(destPath string) error {

	var (
		err  error
		info os.FileInfo
	)

	if !testCookbookBuilt {

		fmt.Printf("\n* Cookbook build destination path: %s\n\n", destPath)

		cleanTest := os.Getenv("CBS_CLEAN_TEST")
		cookbookDestPath := filepath.Join(destPath, "dist")
		info, err = os.Stat(cookbookDestPath)
		if cleanTest == "1" || os.IsNotExist(err) || !info.IsDir() {
			os.RemoveAll(filepath.Join(filepath.Dir(destPath), ".build"))

			err = buildCookbookFixture(
				cookbookDestPath,
				"https://github.com/appbricks/cloud-builder/test/fixtures/recipes",
				"master",
				"basic",
				"google",
				"test", 
				"0.0.1",
				true,
			)
			if err != nil {
				return err
			}
			err = buildCookbookFixture(
				cookbookDestPath,
				"https://github.com/appbricks/cloud-builder/test/fixtures/recipes",
				"master",
				"basic",
				"aws",
				"test", 
				"0.0.1",
				false,
			)
			if err != nil {
				return err
			}
			err = buildCookbookFixture(
				cookbookDestPath,
				"https://github.com/appbricks/cloud-builder/test/fixtures/recipes",
				"master",
				"simple",
				"google",
				"test", 
				"0.0.1",
				false,
			)
			if err != nil {
				return err
			}
			
			// cookbook to import
			importCookbookPath := filepath.Join(destPath, "import")
			err = buildCookbookFixture(
				importCookbookPath,
				"https://github.com/appbricks/minecraft/cloud/recipes",
				"main",
				"",
				"",
				"minecraft", "1.2.3",
				true,
			)
			if err != nil {
				return err
			}			

		} else {
			fmt.Printf("  Cookbook archive exists at destination. It will not be rebuilt.\n\n")
		}

		testCookbookBuilt = true
	}
	return nil
}

// Execute script to create cookbook
// distribution to use as a text fixture
func buildCookbookFixture(
	destPath, 
	repo, 
	branch, 
	name, 
	iaas, 
	cookbookName, 
	cookbookVersion string, 
	clean bool,
) error {

	var (
		err error
		cmd *exec.Cmd

		sourceDirPath,
		cookbookBuildScript string
	)

	_, filename, _, _ := runtime.Caller(0)
	sourceDirPath = path.Dir(filename)

	cookbookBuildScript, err = filepath.Abs(fmt.Sprintf("%s/../../scripts/build-cookbook.sh", sourceDirPath))
	if err != nil {
		ginkgo.Fail(err.Error())
	}

	args := []string{
		"--verbose",
		"--recipe", repo,
		"--git-branch", branch,	
		"--dest-dir", destPath,
	}
	if len(name) > 0 {
		args = append(args, "--name", name)
	}
	if len(iaas) > 0 {
		args = append(args, "--iaas", iaas)
	}
	if len(cookbookName) > 0 {
		args = append(args, "--cookbook-name", cookbookName)
	}
	if len(cookbookVersion) > 0 {
		args = append(args, "--cookbook-version", cookbookVersion)
	}
	if clean {
		args = append(args, "--clean")		
	}

	cmd = exec.Command(cookbookBuildScript, args...)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	fmt.Printf(
		"--> Building cookbook:\n    recipe name: %s\n    iaas: %s\n    repo folder: %s\n\n",
		name, iaas, repo)

	cmd.Dir = os.TempDir()
	err = cmd.Run()
	if err != nil {
		fmt.Printf("%s\n^^^^ ERROR! ^^^^\n\n", out.String())
		return fmt.Errorf(err.Error())
	}

	return nil
}
