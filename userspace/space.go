package userspace

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"

	"github.com/mevansam/goutils/crypto"
	"github.com/mevansam/goutils/logger"
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
	CanUseAsEgressNode() bool

	GetApiCARoot() string
	GetEndpoint() (string, error)
	RestApiClient(ctx context.Context) (*rest.RestApiClient, error)

	CreateDeviceConnectKeyPair() (string, string, error)
}

type Space struct {
	SpaceID     string `json:"spaceID"`
	SpaceName   string `json:"spaceName"`

	PublicKey   string `json:"publicKey"`
	Certificate string `json:"certificate"`
	
	Cookbook string
	Recipe   string
	IaaS     string
	Region   string	
	Version  string

	IsEgressNode bool

	Status    string `json:"status"`
	LastSeen  uint64

	// space access for
	// user in context
	IsOwned      bool
	IsAdmin      bool
	AccessStatus string

	IPAddress   string `json:"ipAddress"`
	FQDN        string `json:"fqdn"`
	Port        int    `json:"port"`
	VpnType     string
	LocalCARoot string `json:"localCARoot"`
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
	return s.SpaceName
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

func (s *Space) CanUseAsEgressNode() bool {
	return s.IsEgressNode
}

func (s *Space) GetApiCARoot() string {
	return s.LocalCARoot
}

func (s *Space) GetEndpoint() (string, error) {

	var (
		protocol string
		host     string
	)

	if s.Port == 443 || len(s.LocalCARoot) > 0 {
		protocol = "https"
	} else {
		protocol = "http"
	}
	if len(s.FQDN) > 0 {
		host = s.FQDN
	} else if len(s.IPAddress) > 0 {
		host = s.IPAddress
	} else {
		return "", fmt.Errorf("unable to determine the api host name for space '%s'", s.SpaceName)
	}
	if s.Port == 0 || s.Port == 80 || s.Port == 443 {
		return fmt.Sprintf("%s://%s", protocol, host), nil 
	}
	return fmt.Sprintf("%s://%s:%d", protocol, host, s.Port), nil
}

func (s *Space) RestApiClient(ctx context.Context) (*rest.RestApiClient, error) {

	var (
		err error

		certPool   *x509.CertPool
		httpClient *http.Client
		
	  endpoint string
	)

	if len(s.LocalCARoot) > 0 {
		if certPool, err = x509.SystemCertPool(); err != nil {
			logger.DebugMessage(
				"Space.RestApiClient(): Using new empty cert pool due to error retrieving system cert pool.: %s", 
				err.Error(),
			)
			certPool = x509.NewCertPool()
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
	if endpoint, err = s.GetEndpoint(); err != nil {
		return nil, err
	}
	return rest.NewRestApiClient(ctx, endpoint).WithHttpClient(httpClient), nil
}

func (s *Space) CreateDeviceConnectKeyPair() (string, string, error) {
	return crypto.CreateVPNKeyPair(s.VpnType)
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
