package auth

import (
	"github.com/appbricks/cloud-builder/config"
	"github.com/appbricks/cloud-builder/userspace"
)

// User Role
type Role uint
const (
	// grants admin access both at 
	// local/device and remote/space
	Admin Role = iota
	// grants access to remote 
	// management space functions
	Manager
	// guest access
	Guest
)
// Space user role mask
type RoleMask uint

func NewRoleFromString(r string) Role {
	switch r {
	case "admin":
		return Admin
	case "manager":
		return Manager
	default:
		return Guest
	}
}

func (r Role) String() string {
	return []string{
		"admin", 
		"manager", 
		"guest",
	}[r]
}

func NewRoleMask(roles... Role) RoleMask {
	mask := 0
	for _, r := range roles {
		mask = mask | (1 << r)
	}
	return RoleMask(mask)
}

func (m RoleMask) HasOnlyRole(r Role) bool {
	rm := uint(1) << r
	return (uint(m) | rm == rm)
}

func (m RoleMask) HasRole(r Role) bool {
	rm := uint(1) << r
	return (uint(m) & rm == rm)
}

// check if user logged into device is 
// authorized using the give mask
func (m RoleMask) LoggedInUserHasRole(
	deviceContext config.DeviceContext, 
	spaceNode userspace.SpaceNode,
) bool {

	ownerUserID, isOwnerConfigured := deviceContext.GetOwnerUserID()
	if isOwnerConfigured && ownerUserID == deviceContext.GetLoggedInUserID() {
		return m.HasRole(Admin)
	}
	if spaceNode != nil && spaceNode.HasAdminAccess() {
		return m.HasRole(Manager)
	}
	return m.HasRole(Guest)
}
