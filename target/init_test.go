package target_test

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mevansam/goutils/logger"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	utils_mocks "github.com/mevansam/goutils/test/mocks"
)

var (
	sourceDirPath,
	workspacePath string
)

func TestCookbook(t *testing.T) {
	logger.Initialize()

	workspacePath = filepath.Join(os.TempDir(), "cbs_workspace")

	_, filename, _, _ := runtime.Caller(0)
	sourceDirPath = path.Dir(filename)

	utils_mocks.Init()

	RegisterFailHandler(Fail)
	RunSpecs(t, "target")
}

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
