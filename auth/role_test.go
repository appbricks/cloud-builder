package auth_test

import (
	"github.com/appbricks/cloud-builder/auth"
	"github.com/appbricks/cloud-builder/config"
	"github.com/appbricks/cloud-builder/userspace"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Authenticator", func() {

	var (
		err error
	)

	It("Creates roles", func() {
		Expect(auth.Admin.String()).To(Equal("admin"))
		Expect(auth.Manager.String()).To(Equal("manager"))
		Expect(auth.Guest.String()).To(Equal("guest"))
		Expect(auth.RoleFromString("admin")).To(Equal(auth.Admin))
		Expect(auth.RoleFromString("manager")).To(Equal(auth.Manager))
		Expect(auth.RoleFromString("guest")).To(Equal(auth.Guest))
	})

	It("Creates roles from context", func() {

		dc := config.NewDeviceContext()
		_, err = dc.NewOwnerUser("ownerUserID", "ownerUserName")
		Expect(err).NotTo(HaveOccurred())
		
		dc.SetLoggedInUser("ownerUserID", "ownerUserName")
		Expect(auth.RoleFromContext(dc, nil)).To(Equal(auth.Admin))
		Expect(auth.RoleFromContext(dc, &userspace.Space{ IsOwned: true, IsAdmin: true })).To(Equal(auth.Admin))
		Expect(auth.RoleFromContext(dc, &userspace.Space{ IsOwned: false, IsAdmin: true })).To(Equal(auth.Manager))
		Expect(auth.RoleFromContext(dc, &userspace.Space{ IsOwned: false, IsAdmin: false })).To(Equal(auth.Guest))
		dc.SetLoggedInUser("userID", "userName")
		Expect(auth.RoleFromContext(dc, nil)).To(Equal(auth.Guest))
		Expect(auth.RoleFromContext(dc, &userspace.Space{ IsAdmin: true })).To(Equal(auth.Manager))
		Expect(auth.RoleFromContext(dc, &userspace.Space{ IsAdmin: false })).To(Equal(auth.Guest))
	})

	It("Validates roles against a role mask", func() {
		
		roleMask := auth.NewRoleMask(auth.Admin, auth.Guest)
		Expect(roleMask.HasRole(auth.Admin)).To(BeTrue())
		Expect(roleMask.HasRole(auth.Guest)).To(BeTrue())
		Expect(roleMask.HasOnlyRole(auth.Admin)).To(BeFalse())
		Expect(roleMask.HasOnlyRole(auth.Guest)).To(BeFalse())

		dc := config.NewDeviceContext()
		_, err = dc.NewOwnerUser("ownerUserID", "ownerUserName")
		Expect(err).NotTo(HaveOccurred())
		dc.SetLoggedInUser("ownerUserID", "ownerUserName")
		Expect(roleMask.LoggedInUserHasRole(dc, nil)).To(BeTrue())

		roleMask = auth.NewRoleMask(auth.Admin)
		Expect(roleMask.HasOnlyRole(auth.Admin)).To(BeTrue())
		Expect(roleMask.HasOnlyRole(auth.Guest)).To(BeFalse())
		Expect(roleMask.HasRole(auth.Admin)).To(BeTrue())
		Expect(roleMask.HasRole(auth.Guest)).To(BeFalse())
	})
})