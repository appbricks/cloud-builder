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

func RoleFromString(r string) Role {
	switch r {
	case "admin":
		return Admin
	case "manager":
		return Manager
	default:
		return Guest
	}
}

func RoleFromContext(
	deviceContext config.DeviceContext, 
	spaceNode userspace.SpaceNode,
) Role {
	//
	// Admin - user logged in to device is also the device and space owner. 
	//
	// Manager - user logged in to device IS the device owner, but has been 
	//           granted admin access to the space. the user logged in to 
	//           device IS NOT the device owner and has admin access to the 
	//           space. space owners are also space admins by default so the
	//           user that falls into this role could be the space owner.
	//
	// Guest - all other users that have access to the space
	//  
	ownerUserID, isOwnerConfigured := deviceContext.GetOwnerUserID()
	if isOwnerConfigured && ownerUserID == deviceContext.GetLoggedInUserID() {
		if spaceNode == nil || spaceNode.IsSpaceOwned() {
			return Admin
		}
	}
	if spaceNode != nil && spaceNode.HasAdminAccess() {
		return Manager
	}
	return Guest
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
	return m.HasRole(
		RoleFromContext(
			deviceContext,
			spaceNode,
		),
	)
}
