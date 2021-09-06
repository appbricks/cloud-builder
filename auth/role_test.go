package auth_test

import (
	"github.com/appbricks/cloud-builder/auth"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Authenticator", func() {

	It("Creates roles", func() {
		Expect(auth.Admin.String()).To(Equal("admin"))
		Expect(auth.Manager.String()).To(Equal("manager"))
		Expect(auth.Guest.String()).To(Equal("guest"))
		Expect(auth.NewRoleFromString("admin")).To(Equal(auth.Admin))
		Expect(auth.NewRoleFromString("manager")).To(Equal(auth.Manager))
		Expect(auth.NewRoleFromString("guest")).To(Equal(auth.Guest))
	})

	It("Validates roles against a role mask", func() {
		
		roleMask := auth.NewRoleMask(auth.Admin, auth.Guest)
		Expect(roleMask.HasRole(auth.Admin)).To(BeTrue())
		Expect(roleMask.HasRole(auth.Guest)).To(BeTrue())
		Expect(roleMask.HasOnlyRole(auth.Admin)).To(BeFalse())
		Expect(roleMask.HasOnlyRole(auth.Guest)).To(BeFalse())

		roleMask = auth.NewRoleMask(auth.Admin)
		Expect(roleMask.HasOnlyRole(auth.Admin)).To(BeTrue())
		Expect(roleMask.HasOnlyRole(auth.Guest)).To(BeFalse())
		Expect(roleMask.HasRole(auth.Admin)).To(BeTrue())
		Expect(roleMask.HasRole(auth.Guest)).To(BeFalse())
	})
})