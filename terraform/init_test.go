package terraform_test

import (
	"path"
	"runtime"
	"testing"

	"github.com/mevansam/goutils/logger"
	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	utils_mocks "github.com/mevansam/goutils/test/mocks"
)

var (
	sourceDirPath string
)

func TestCookbook(t *testing.T) {
	logger.Initialize()

	_, filename, _, _ := runtime.Caller(0)
	sourceDirPath = path.Dir(filename)

	utils_mocks.Init()

	RegisterFailHandler(Fail)
	RunSpecs(t, "terraform")
}

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
