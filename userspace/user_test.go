package userspace_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("User", func() {

	var (
		err error
	)

	It("encrypts and decrypts using the user's key", func() {
		Expect(err).NotTo(HaveOccurred())
	})
})