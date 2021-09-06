package userspace

import (
	"context"
	"strings"

	"github.com/mevansam/goutils/rest"
)

type SpaceNode interface {

	GetSpaceID() string
	GetSpaceName() string
	GetPublicKey() string

	GetRecipe() string
	GetIaaS() string
	GetRegion() string
	GetVersion() string

	GetStatus() string
	GetLastSeen() uint64

	IsRunning() bool

	HasAdminAccess() bool

	RestApiClient(ctx context.Context) (*rest.RestApiClient, error)
}

type Space struct {
	SpaceID   string
	SpaceName string
	PublicKey string
	
	Recipe  string
	IaaS    string
	Region  string	
	Version string

	Status    string
	LastSeen  uint64

	// space access for
	// user in context
	IsAdmin      bool
	AccessStatus string

	IPAddress   string
	FQDN        string
	Port        int
	LocalCARoot string
}

type SpaceUser struct {
	UserID string `json:"userID"`
	Name   string `json:"name"`

	IsOwner bool `json:"isOwner"`
	IsAdmin bool `json:"isAdmin"`

	// active devices for this users
	Devices []*Device `json:"devices,omitempty"`
}

func (s *Space) GetSpaceID() string {
	return s.SpaceID
}

func (s *Space) GetSpaceName() string {
	return s.SpaceName
}

func (s *Space) GetPublicKey() string {
	return s.PublicKey
}

func (s *Space) GetRecipe() string {
	return s.Recipe
}

func (s *Space) GetIaaS() string {
	return s.IaaS
}

func (s *Space) GetRegion() string {
	return s.Region
}

func (s *Space) GetVersion() string {
	return s.Version
}

func (s *Space) GetStatus() string {
	if len(s.Status) == 0 {
		return "unknown"
	}
	return s.Status
}

func (s *Space) GetLastSeen() uint64 {
	return s.LastSeen
}

func (s *Space) IsRunning() bool {
	return s.Status == "running"
}

func (s *Space) HasAdminAccess() bool {
	return s.IsAdmin
}

func (s *Space) RestApiClient(ctx context.Context) (*rest.RestApiClient, error) {
	return nil, nil
}

// sorter to order spaces in order of recipe, cloud, region and deployment name
type SpaceCollection []*Space

func (sc SpaceCollection) Len() int {
	return len(sc)
}

func (sc SpaceCollection) Less(i, j int) bool {
	s1 := sc[i]
	s2 := sc[j]

	return strings.Compare(s1.Recipe, s2.Recipe) == -1 ||
		( strings.Compare(s1.Recipe, s2.Recipe) == 0 && 
			strings.Compare(s1.IaaS, s2.IaaS) == -1 ) || 
		( strings.Compare(s1.Recipe, s2.Recipe) == 0 && 
			strings.Compare(s1.IaaS, s2.IaaS) == 0 && 
			strings.Compare(s1.Region, s2.Region) == -1 ) || 
		( strings.Compare(s1.Recipe, s2.Recipe) == 0 && 
			strings.Compare(s1.IaaS, s2.IaaS) == 0 && 
			strings.Compare(s1.Region, s2.Region) == 0 &&
			strings.Compare(s1.SpaceName, s2.SpaceName) == -1 )
}

func (sc SpaceCollection) Swap(i, j int) {
	s := sc[i]
	sc[i] = sc[j]
	sc[j] = s
}
