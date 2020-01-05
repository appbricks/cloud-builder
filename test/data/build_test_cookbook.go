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
		info, err = os.Stat(filepath.Join(destPath, "dist"))
		if cleanTest == "1" || os.IsNotExist(err) || !info.IsDir() {

			err = buildCookbookFixture(
				destPath,
				"https://github.com/appbricks/cloud-builder/test/fixtures/recipes",
				"basic",
				"google",
				true,
			)
			if err != nil {
				return err
			}
			err = buildCookbookFixture(
				destPath,
				"https://github.com/appbricks/cloud-builder/test/fixtures/recipes",
				"basic",
				"aws",
				false,
			)
			if err != nil {
				return err
			}
			err = buildCookbookFixture(
				destPath,
				"https://github.com/appbricks/cloud-builder/test/fixtures/recipes",
				"simple",
				"google",
				false,
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
func buildCookbookFixture(destPath, repo, name, iaas string, clean bool) error {

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

	if clean {
		cmd = exec.Command(cookbookBuildScript,
			"--verbose",
			"--clean",
			"--recipe", repo,
			"--name", name,
			"--iaas", iaas)
	} else {
		cmd = exec.Command(cookbookBuildScript,
			"--verbose",
			"--recipe", repo,
			"--name", name,
			"--iaas", iaas)
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	fmt.Printf(
		"--> Building cookbook:\n    recipe name: %s\n    iaas: %s\n    repo folder: %s\n\n",
		name, iaas, repo)

	cmd.Dir = os.TempDir()
	cmd.Env = append(os.Environ(),
		"HOME_DIR="+destPath)

	err = cmd.Run()
	if err != nil {
		fmt.Printf("%s\n^^^^ ERROR! ^^^^\n\n", out.String())
		return fmt.Errorf(err.Error())
	}

	return nil
}
