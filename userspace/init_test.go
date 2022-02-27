package userspace_test

import (
	"testing"

	"github.com/mevansam/goutils/logger"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestUserspace(t *testing.T) {
	logger.Initialize()

	RegisterFailHandler(Fail)
	RunSpecs(t, "userspace")
}

var _ = AfterSuite(func() {
})
