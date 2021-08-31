package auth

// User Role
type Role uint
const (
	Admin Role = iota
	Guest
) 
// Space user role mask
type RoleMask uint

func NewRoleFromString(r string) Role {
	switch r {
	case "admin":
		return Admin
	default:
		return Guest
	}
}

func (r Role) String() string {
	return []string{"admin", "guest"}[r]
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
