package config_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/mevansam/goutils/logger"
	"github.com/onsi/gomega/gexec"
)

var (
	workspacePath string
)

func TestConfig(t *testing.T) {
	logger.Initialize()

	workspacePath = filepath.Join(os.TempDir(), "cbs_workspace")

	RegisterFailHandler(Fail)
	RunSpecs(t, "config")
}

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
