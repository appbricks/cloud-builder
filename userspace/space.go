package userspace

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"

	"github.com/mevansam/goutils/rest"
)

type SpaceNode interface {

	// a key for the space node
	Key() string

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

	IsSpaceOwned() bool
	HasAdminAccess() bool

	RestApiClient(ctx context.Context) (*rest.RestApiClient, error)
}

type Space struct {
	key string

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
	IsOwned      bool
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

func (s *Space) Key() string {

	var (
		key strings.Builder
	)

	if len(s.key) == 0 {
		key.WriteString(s.Recipe)
		key.Write([]byte{'/'})
		key.WriteString(s.IaaS)
		key.Write([]byte{'/'})
		key.WriteString(s.Region)
		key.Write([]byte{'/'})
		key.WriteString(s.SpaceName)
		s.key = key.String()
	}
	return s.key
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

func (s *Space) IsSpaceOwned() bool {
	return s.IsOwned
}

func (s *Space) HasAdminAccess() bool {
	return s.IsAdmin
}

func (s *Space) RestApiClient(ctx context.Context) (*rest.RestApiClient, error) {

	var (
		err error

		certPool   *x509.CertPool
		httpClient *http.Client
		
		host string
		url  string
	)

	if len(s.LocalCARoot) > 0 {
		if certPool, err = x509.SystemCertPool(); err != nil {
			return nil, err
		}
		certPool.AppendCertsFromPEM([]byte(s.LocalCARoot))
	
		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: certPool,
				},
			},
		}
	} else {
		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{},
			},
		}
	}
	if len(s.FQDN) > 0 {
		host = s.FQDN
	} else if len(s.IPAddress) > 0 {
		host = s.IPAddress
	} else {
		return nil, fmt.Errorf("unable to determine the api host name for space '%s'", s.SpaceName)
	}
	if s.Port > 0 {
		url = fmt.Sprintf("https://%s:%d", host, s.Port)
	} else {
		url = fmt.Sprintf("https://%s", host)
	}
	return rest.NewRestApiClient(ctx, url).WithHttpClient(httpClient), nil
}

// sorter to order spaces in order of recipe, cloud, region and deployment name
type SpaceCollection []SpaceNode

func (sc SpaceCollection) Len() int {
	return len(sc)
}

func (sc SpaceCollection) Less(i, j int) bool {

	var (
		recipeComp, 
		iaasComp, 
		regionComp,
		spaceNameComp int
	)
	s1 := sc[i]
	s2 := sc[j]

	recipe1 := s1.GetRecipe()
	recipe2 := s2.GetRecipe()
	if recipeComp = strings.Compare(recipe1, recipe2); 
		recipeComp == -1 {
		return true
	}
	iaas1 := s1.GetIaaS()
	iaas2 := s2.GetIaaS()
	if iaasComp = strings.Compare(iaas1, iaas2); 
		recipeComp == 0 && iaasComp == -1 {
		return true
	}
	region1 := s1.GetRegion()
	region2 := s2.GetRegion()
	if regionComp = strings.Compare(region1, region2); 
		recipeComp == 0 && iaasComp == 0 && regionComp == -1 {
		return true
	}
	spaceName1 := s1.GetSpaceName()
	spaceName2 := s2.GetSpaceName()
	if spaceNameComp = strings.Compare(spaceName1, spaceName2); 
		recipeComp == 0 && iaasComp == 0 && regionComp == 0 && spaceNameComp == -1 {
		return true
	}
	return false
}

func (sc SpaceCollection) Swap(i, j int) {
	s := sc[i]
	sc[i] = sc[j]
	sc[j] = s
}
