package target

import (
	pcontext "context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/appbricks/cloud-builder/cookbook"
	"github.com/appbricks/cloud-builder/terraform"
	"github.com/mevansam/gocloud/backend"
	"github.com/mevansam/gocloud/cloud"
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/config"
	"github.com/mevansam/goutils/crypto"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/rest"
)

// Instance state callback
type InstanceStateChange func(name string, instance *ManagedInstance)

// Input types
type TargetState int

const (
	Undeployed TargetState = iota
	Running
	Shutdown
	Pending
	Unknown
)

// a target is a recipe configured to be
// launched in a public cloud region
type Target struct {
	RecipeName   string `json:"recipeName"`
	RecipeIaas   string `json:"recipeIaas"`

	CookbookName    string `json:"cookbookName,omitempty"`
	CookbookVersion string `json:"cookbookVersion,omitempty"`

	DependentTargets []string `json:"dependentTargets"`

	Recipe   cookbook.Recipe        `json:"recipe,omitempty"`
	Provider provider.CloudProvider `json:"provider,omitempty"`
	Backend  backend.CloudBackend   `json:"backend,omitempty"`

	Output *map[string]terraform.Output `json:"output,omitempty"`

	RSAPrivateKey string `json:"rsaPrivateKey,omitempty"`
	RSAPublicKey  string `json:"rsaPublicKey,omitempty"`

	NodeKey string `json:"nodeKey,omitempty"`
	NodeID  string `json:"nodeID,omitempty"`

	dependencies []*Target
	dependents int

	description string
	version     string

	rootCACert string
	vpnType    string

	managedInstances []*ManagedInstance
	compute          cloud.Compute

	loadingState       int
	loadRemoteRefWG    sync.WaitGroup
	loadRemoteRefError error
}

type loadingStates = int
const (
	dirty loadingStates = iota
	loading
	loaded
)

type ManagedInstance struct {
	Instance cloud.ComputeInstance
	Metadata map[string]interface{}

	order int

	// recipe state output values
	// these can be IaaS specific
	id,
	name,
	description,
	fqdn,
	publicIP,
	privateIP,
	hcPort,
	hcType,
	apiPort,
	sshPort,
	sshUser,
	sshKey,
	rootUser,
	rootPasswd,
	nonRootUser,
	nonRootPasswd,
	rootCACert string
}

// create a target key
func CreateKey(
	recipeKey, iaasName string, 
	keyValues ...string,
) string {
	
	var (
		key strings.Builder
	)
	
	key.WriteString(recipeKey)
	key.Write([]byte{'/'})
	key.WriteString(iaasName)
	key.Write([]byte{'/'})
	key.WriteString(strings.Join(keyValues, "/"))
	return key.String()
}

func NewTarget(
	r cookbook.Recipe, 
	p provider.CloudProvider, 
	b backend.CloudBackend,
) *Target {

	return &Target{
		RecipeName: r.RecipeName(),
		RecipeIaas: p.Name(),

		CookbookName:    r.CookbookName(),
		CookbookVersion: r.CookbookVersion(),

		DependentTargets: []string{},

		Recipe:   r,
		Provider: p,
		Backend:  b,

		dependencies: []*Target{},
	}
}

func (t *Target) Name() string {
	region := t.Provider.Region()

	if region == nil {
		return fmt.Sprintf(
			"Deployment \"%s\" on \"%s\".",
			t.DeploymentName(),
			t.Provider.Name(),
		)
	
	} else {
		return fmt.Sprintf(
			"Deployment \"%s\" on Cloud \"%s\" and Region \"%s\"",
			t.DeploymentName(),
			t.Provider.Name(),
			*region,
		)	
	}
}

func (t *Target) Description() string {
	return t.description
}

func (t *Target) Version() string {
	return t.version
}

func (t *Target) DeploymentName() string {

	if variable, exists := t.Recipe.GetVariable("name"); exists && variable.Value != nil {
		return *variable.Value
	} else {
		return "NONAME"
	}
}

func (t *Target) Dependencies() []*Target {
	return t.dependencies
}

func (t *Target) HasDependents() bool {
	return t.dependents > 0
}

func (t *Target) UpdateKeys() (*Target, error) {

	var (
		err error
	)

	// create new target key pair
	if t.RSAPrivateKey, t.RSAPublicKey, err = crypto.CreateRSAKeyPair(nil); err != nil {
		return nil, err
	}	
	return t, nil
}

// functions referencing target's remote managed cloud instances

func (t *Target) ManagedInstances() []*ManagedInstance {

	if t.loadingState != loading && 
		(t.loadingState == dirty || t.managedInstances == nil) {
		t.Refresh()
	}
	t.loadRemoteRefWG.Wait()
	return t.managedInstances
}

func (t *Target) ManagedInstance(name string) *ManagedInstance {

	for _, managedInstance := range t.ManagedInstances() {
		if managedInstance.name == name {
			return managedInstance
		}
	}
	return nil
}

func (t *Target) Resume(cb InstanceStateChange) error {

	var (
		err error
	)

	if t.Status() == Shutdown {
		for _, managedInstance := range t.managedInstances {
			cb(managedInstance.name, managedInstance)
			if err = managedInstance.Instance.Start(); err != nil {
				return err
			}
			cb(managedInstance.name, managedInstance)
		}
	} else {
		return fmt.Errorf("target is not in a 'shutdown' state")
	}

	return nil
}

func (t *Target) Suspend(cb InstanceStateChange) error {

	var (
		err error
	)

	if t.Status() == Running {
		for _, managedInstance := range t.managedInstances {
			cb(managedInstance.name, managedInstance)
			if err = managedInstance.Instance.Stop(); err != nil {
				return err
			}
			cb(managedInstance.name, managedInstance)
		}
	} else {
		return fmt.Errorf("target is not in a 'shutdown' state")
	}

	return nil
}

func (t *Target) Status() TargetState {

	var (
		err error

		state cloud.InstanceState
	)

	if t.Output != nil {
		managedInstances := t.ManagedInstances()
		if managedInstances == nil || t.loadRemoteRefError != nil {
			return Unknown
		}

		numInstances := len(managedInstances)
		numRunning := 0
		numStopped := 0
		numPending := 0
		numUnknown := 0
		for _, instance := range managedInstances {
			if state, err = instance.State(); err != nil {
				logger.TraceMessage(
					"Managed instance '%s' of target '%s' returned an error when querying its state: %s",
					instance.Instance.Name(), t.Key(), err.Error(),
				)
				numUnknown++
				continue
			}	
			switch state {
			case cloud.StateRunning:
				numRunning++
			case cloud.StateStopped:
				numStopped++
			case cloud.StatePending:
				numPending++
			default:
				numUnknown++
			}
		}

		if numUnknown == numInstances {
			return Unknown
		}
		if numRunning == numInstances {
			return Running
		}
		if numStopped == numInstances {
			return Shutdown
		}
		if (numRunning + numStopped + numPending) == numInstances {
			return Pending
		}

		logger.DebugMessage(
			"Unable to determine state of target '%s'. Have %d instances with state %d running, %d stopped, %d pending and %d unknown.",
			t.Key(), numInstances, numRunning, numStopped, numPending, numUnknown,
		)
		return Unknown

	} else {
		return Undeployed
	}
}

func (t *Target) Error() error {
	t.loadRemoteRefWG.Wait()
	return t.loadRemoteRefError
}

func (t *Target) Refresh() {
	
	t.loadingState = loading
	t.loadRemoteRefWG.Add(1)
	
	go func() {

		defer func() {
			t.loadingState = loaded
		}()

		if t.loadRemoteRefError = t.loadRemoteRefs(); t.loadRemoteRefError != nil {
			logger.DebugMessage(
				"Error refreshing remote refs of target '%s':", 
				t.Key(), t.loadRemoteRefError.Error())
		}
	}()
}

// load target cloud references
func (t *Target) loadRemoteRefs() error {

	var (
		err error
		ok  bool

		output terraform.Output

		managedInstanceValues []interface{}
		instanceMetaData      map[string]interface{}

		instance       *ManagedInstance
		cloudInstances []cloud.ComputeInstance

		value    interface{}
		order    float64
		keyValue string

		instanceRef map[string]*ManagedInstance
	)

	defer t.loadRemoteRefWG.Done()

	readKeyValue := func(key string) (string, error) {
		if value, ok = instanceMetaData[key]; !ok {
			return "",
				fmt.Errorf("managed instance metadata did no contain a '%s' key", key)
		}
		if keyValue, ok = value.(string); !ok {
			return "",
				fmt.Errorf("managed instance metadata '%s' key value is not a string", key)
		}
		return keyValue, nil
	}

	if t.compute == nil {
		logger.TraceMessage("Connecting to provider '%s'.", t.Provider.Name())
		if err = t.Provider.Connect(); err != nil {
			return err
		}
		if t.compute, err = t.Provider.GetCompute(); err != nil {
			return err
		}
	}
	if t.Output != nil {
		logger.TraceMessage("Target deployment output: %# v", t.Output)

		if output, ok = (*t.Output)["cb_node_description"]; ok {
			if t.description, ok = output.Value.(string); !ok {
				return fmt.Errorf("node description key value is not a string")
			}
		}
		if output, ok = (*t.Output)["cb_node_version"]; ok {
			if t.version, ok = output.Value.(string); !ok {
				return fmt.Errorf("node version key value is not a string")
			}
		}
		if output, ok = (*t.Output)["cb_root_ca_cert"]; ok {
			if t.rootCACert, ok = output.Value.(string); !ok {
				return fmt.Errorf("node root ca certificate key value is not a string")
			}
		}
		if output, ok = (*t.Output)["cb_vpn_type"]; ok {
			if t.vpnType, ok = output.Value.(string); !ok {
				return fmt.Errorf("node root vpn type value is not a string")
			}
		}

		if output, ok = (*t.Output)["cb_managed_instances"]; ok {

			if managedInstanceValues, ok = output.Value.([]interface{}); !ok {
				return fmt.Errorf("managed instance output is not a list")
			}

			numInstance := len(managedInstanceValues)
			t.managedInstances = make([]*ManagedInstance, 0, numInstance)

			ids := make([]string, numInstance)
			instanceRef = make(map[string]*ManagedInstance)

			for i, managedInstanceValue := range managedInstanceValues {
				if instanceMetaData, ok = managedInstanceValue.(map[string]interface{}); !ok {
					return fmt.Errorf("managed instance metadata value is not a map of key value pairs")
				}

				instance = &ManagedInstance{
					Metadata:   instanceMetaData,
					order:      math.MaxInt64,
					rootCACert: t.rootCACert,
				}
				if value, ok = instanceMetaData["order"]; ok {
					if order, ok = value.(float64); !ok {
						return fmt.Errorf("managed instance metadata name key value is not a string")
					}
					instance.order = int(order)
				}
				if instance.id, err = readKeyValue("id"); err != nil {
					return err
				}
				if instance.name, err = readKeyValue("name"); err != nil {
					return err
				}
				if instance.description, err = readKeyValue("description"); err != nil {
					return err
				}
				if instance.fqdn, err = readKeyValue("fqdn"); err != nil {
					return err
				}
				if instance.publicIP, err = readKeyValue("public_ip"); err != nil {
					return err
				}
				if instance.privateIP, err = readKeyValue("private_ip"); err != nil {
					return err
				}
				if instance.hcPort, err = readKeyValue("health_check_port"); err != nil {
					return err
				}
				if instance.hcType, err = readKeyValue("health_check_type"); err != nil {
					return err
				}
				if instance.apiPort, err = readKeyValue("api_port"); err != nil {
					return err
				}
				if instance.sshPort, err = readKeyValue("ssh_port"); err != nil {
					return err
				}
				if instance.sshUser, err = readKeyValue("ssh_user"); err != nil {
					return err
				}
				if instance.sshKey, err = readKeyValue("ssh_key"); err != nil {
					return err
				}
				if instance.rootUser, err = readKeyValue("root_user"); err != nil {
					return err
				}
				if instance.rootPasswd, err = readKeyValue("root_passwd"); err != nil {
					return err
				}
				if instance.nonRootUser, err = readKeyValue("non_root_user"); err != nil {
					return err
				}
				if instance.nonRootPasswd, err = readKeyValue("non_root_passwd"); err != nil {
					return err
				}

				ids[i] = instance.id
				instanceRef[ids[i]] = instance

				// insert instance into managed instance list in order
				j := sort.Search(i, func(j int) bool {
					managedInstance := t.managedInstances[j]
					return managedInstance.order > instance.order ||
						(managedInstance.order == instance.order &&
							strings.Compare(managedInstance.name, instance.name) == 1)
				})
				t.managedInstances = append(t.managedInstances, instance)
				if len(t.managedInstances) > 1 {
					copy(t.managedInstances[i+1:], t.managedInstances[i:])
					t.managedInstances[j] = instance
				}
			}

			if t.compute != nil {
				logger.TraceMessage("Retrieving managed instances: %# v", ids)
				if cloudInstances, err = t.compute.GetInstances(ids); err != nil {
					return err
				}	
				if len(cloudInstances) == 0 {
					return fmt.Errorf("unable to lookup managed instances from cloud provider")
				}
				for _, cloudInstance := range cloudInstances {
					instanceRef[cloudInstance.ID()].Instance = cloudInstance
				}
			} else {
				logger.TraceMessage(
					"Provider '%s' does not have a compute backend. Instances in deployment will default to unmanaged: %# v", 
					t.Provider.Name(), ids,
				)
			}

		} else {
			logger.DebugMessage(
				"Target '%s' does not appear have any managed instances.",
				t.Key(),
			)
		}
	}

	if t.managedInstances == nil {
		// either target has no managed instances 
		// or it has not been deployed yet
		t.managedInstances = []*ManagedInstance{}
	}

	return nil
}

// returns a copy of this target
func (t *Target) Copy() (*Target, error) {

	var (
		err error

		recipeCopy,
		providerCopy,
		backendCopy config.Configurable
	)

	if recipeCopy, err = t.Recipe.Copy(); err != nil {
		return nil, err
	}

	tgt := &Target{
		RecipeName: t.RecipeName,
		RecipeIaas: t.RecipeIaas,

		CookbookName: t.CookbookName,
		CookbookVersion: t.CookbookVersion,

		DependentTargets: t.DependentTargets,

		Recipe: recipeCopy.(cookbook.Recipe),

		Output: t.Output,

		RSAPrivateKey: t.RSAPrivateKey,
		RSAPublicKey: t.RSAPublicKey,

		NodeKey: t.NodeKey,
		NodeID: t.NodeID,

		dependencies: t.dependencies,
		dependents: t.dependents,

	}
	if t.Provider != nil {
		if providerCopy, err = t.Provider.Copy(); err != nil {
			return nil, err
		}
		tgt.Provider = providerCopy.(provider.CloudProvider)
	}
	if t.Backend != nil {
		if backendCopy, err = t.Backend.Copy(); err != nil {
			return nil, err
		}
		tgt.Backend = backendCopy.(backend.CloudBackend)
	}

	return tgt, nil
}

// prepares the target backend
func (t *Target) PrepareBackend() error {

	var (
		err error

		storage cloud.Storage
	)

	if t.Backend != nil {		
		if !t.Backend.IsValid() {
			return fmt.Errorf(
				"the backend configuration for target %s is not valid",
				t.Key(),
			)
		}
		backEndProviderType := t.Backend.GetProviderType()
		if len(backEndProviderType) > 0 {
			if t.Provider == nil || backEndProviderType != t.Provider.Name() {
				return fmt.Errorf("no provider in recipe available for backend")
			}
			if err = t.Provider.Connect(); err != nil {
				return err
			}
			if storage, err = t.Provider.GetStorage(); err != nil {
				return err
			}
			_, err = storage.NewInstance(t.Backend.GetStorageInstanceName())	
		}
	}
	return err
}

// deletes the target backend
func (t *Target) DeleteBackend() error {

	var (
		err error

		storage  cloud.Storage
		instances []cloud.StorageInstance

		objects []string
	)

	if t.Backend != nil {		
		if !t.Backend.IsValid() {
			return fmt.Errorf(
				"the backend configuration for target %s is not valid",
				t.Key(),
			)
		}
		backEndProviderType := t.Backend.GetProviderType()
		if len(backEndProviderType) > 0 {
			if t.Provider == nil || backEndProviderType != t.Provider.Name() {
				return fmt.Errorf("no provider in recipe available for backend")
			}
			if err = t.Provider.Connect(); err == nil {
				if storage, err = t.Provider.GetStorage(); err == nil {
					if instances, err = storage.ListInstances(); err == nil {
						for _, instance := range instances {
							if instance.Name() == t.Backend.GetStorageInstanceName() {
								if objects, err = instance.ListObjects(""); err != nil {
									return err
								}
								for _, o := range objects {
									if err = instance.DeleteObject(o); err != nil {
										return err
									}
								}
								err = instance.Delete()
								break		
							}
						}
					}
				}
			}
		}
	}
	return err
}

// returns a launcher for this target
func (t *Target) NewBuilder(
	buildVars map[string]string,
	outputBuffer, 
	errorBuffer io.Writer,
) (*Builder, error) {

	if t.Recipe.IsBastion() {
		buildVars["mycs_node_private_key"] = t.RSAPrivateKey
		buildVars["mycs_node_id_key"] = t.NodeKey
	} else {
		buildVars["mycs_app_private_key"] = t.RSAPrivateKey
		buildVars["mycs_app_id_key"] = t.NodeKey
	}
	t.Recipe.AddEnvVars(buildVars)

	for _, dt := range t.dependencies {
		for name, output := range *dt.Output {
			if name != "cb_managed_instances" {
				
				switch v := output.Value.(type) {
				case bool:
					buildVars[name] = strconv.FormatBool(v)
				case int:
					buildVars[name] = strconv.Itoa(v)
				case string:
					buildVars[name] = v
				default:
					b, err := json.Marshal(v)
					if err != nil {
						return nil, err
					}
					buildVars[name] = string(b)
				}
			}
		}
	}

	return NewBuilder(
		strings.Join(t.Recipe.GetKeyFieldValues(), "/"),
		t.Recipe,
		t.Provider,
		t.Backend,
		buildVars,
		outputBuffer,
		errorBuffer)
}

// Target type's SpaceNode implementation

func (t *Target) Key() string {
	
	keyValues := t.Recipe.GetKeyFieldValues()
	for _, dt := range t.dependencies {
		keyValues = append(keyValues, "<"+dt.Key())
	}
	return CreateKey(t.CookbookName + ":" + t.RecipeName, t.RecipeIaas, keyValues...)
}

func (t *Target) GetSpaceID() string {
	return t.NodeID
}

func (t *Target) GetSpaceName() string {
	return t.DeploymentName()
}

func (t *Target) GetPublicKey() string {
	return t.RSAPublicKey
}

func (t *Target) GetRecipe() string {
	return t.RecipeName
}

func (t *Target) GetIaaS() string {
	return t.RecipeIaas
}

func (t *Target) GetRegion() string {
	return *t.Provider.Region()
}

func (t *Target) GetVersion() string {
	return t.Version()
}

func (t *Target) GetStatus() string {

	// force refresh
	t.loadingState = dirty

	return []string{
		"undeployed",
		"running", 
		"shutdown",
		"pending",
		"unknown",
	}[t.Status()]
}

func (t *Target) GetLastSeen() uint64 {
	return 0
}

func (t *Target) IsRunning() bool {

	var (
		err error

		instance      *ManagedInstance
		instanceState cloud.InstanceState
	)

	if t.Status() == Running {
		if instance = t.ManagedInstance("bastion"); instance == nil {
			logger.DebugMessage("Target.isRunning(): Target does not have a managed bastion instance.")
			return false			
		}
		if instanceState, err = instance.State(); err != nil {
			logger.DebugMessage("Target.isRunning(): ERROR! %s", err.Error())
			return false
		}
		return instanceState == cloud.StateRunning
	}
	return false
}

func (t *Target) IsSpaceOwned() bool {
	return true
}

func (t *Target) HasAdminAccess() bool {
	return true
}

func (t *Target) CanUseAsEgressNode() bool {
	return true
}

func (t *Target) GetApiCARoot() string {
	if instance := t.ManagedInstance("bastion"); instance != nil {
		return instance.rootCACert
	}
	return ""
}

func (t *Target) GetEndpoint() (string, error) {

	if instance := t.ManagedInstance("bastion"); instance != nil {
		return instance.GetEndpoint()
	}
	return "", fmt.Errorf("unable to determine endpoint url for target bastion instance")
}

func (t *Target) RestApiClient(ctx pcontext.Context) (*rest.RestApiClient, error) {

	var (
		err error

		instance   *ManagedInstance
		httpClient *http.Client
		url        string
	)

	if instance = t.ManagedInstance("bastion"); instance == nil {
		return nil, fmt.Errorf("target does not have a managed bastion instance")
	}
	if httpClient, url, err = instance.HttpsClient(); err != nil {
		return nil, err
	}
	return rest.NewRestApiClient(ctx, url).WithHttpClient(httpClient), nil
}

func (t *Target) CreateDeviceConnectKeyPair() (string, string, error) {
	return crypto.CreateVPNKeyPair(t.vpnType)
}

// managedInstance functions

func (i *ManagedInstance) Name() string {
	return i.name
}

func (i *ManagedInstance) Description() string {
	return i.description
}

func (i *ManagedInstance) PublicIP() string {
	return i.Instance.PublicIP()
}

func (i *ManagedInstance) FQDN() string {
	return i.fqdn
}

func (i *ManagedInstance) SSHAddress() string {
	
	var (
		publicIP string
	)

	if len(i.publicIP) > 0 {
		// get actual public IP (in cases where managed 
		// instance is deployed using dynamic IP allocation)
		if publicIP = i.Instance.PublicIP(); len(publicIP) == 0 {
			publicIP = i.publicIP
		}
	}
	if len(publicIP) > 0 {
		return fmt.Sprintf("%s:%s", publicIP, i.sshPort)
	} else {
		return fmt.Sprintf("%s:%s", i.privateIP, i.sshPort)
	}
}

func (i *ManagedInstance) SSHUser() string {
	return i.sshUser
}

func (i *ManagedInstance) SSHKey() []byte {
	return []byte(i.sshKey)
}

func (i *ManagedInstance) RootUser() string {
	return i.rootUser
}

func (i *ManagedInstance) RootPassword() string {
	return i.rootPasswd
}

func (i *ManagedInstance) NonRootUser() string {
	return i.nonRootUser
}

func (i *ManagedInstance) NonRootPassword() string {
	return i.nonRootPasswd
}

func (i *ManagedInstance) GetEndpoint() (string, error) {

	var (
		protocol,
		host,
		endpoint string
	)

	if i.apiPort == "443" || len(i.rootCACert) > 0 {
		protocol = "https"
	} else {
		protocol = "http"
	}
	host = i.Instance.PublicDNS()
	if (len(host) == 0) {
		host = fmt.Sprintf(
			"%s.mycs.appbricks.org", strings.ReplaceAll(i.Instance.PublicIP(), ".", "-"),
		)
	}
	if len(host) == 0 {
		if len(i.fqdn) > 0 {
			host = i.fqdn
		} else if len(i.publicIP) > 0 {
			host = i.publicIP
		} else if len(i.privateIP) > 0 {
			host = i.privateIP
		} else {
			return "", fmt.Errorf("unable to determine the managed instance's external host name/ip")
		}	
	}
	if i.apiPort == "0" || i.apiPort == "80" || i.apiPort == "443" {
		endpoint = fmt.Sprintf("%s://%s", protocol, host)
	} else {
		endpoint = fmt.Sprintf("%s://%s:%s", protocol, host, i.apiPort)
	}
	logger.TraceMessage("ManagedInstance.GetEndpoint(): Endpoint for target instance \"%s\" is \"%s\".", i.name, endpoint)
	return endpoint, nil
}

func (i *ManagedInstance) HttpsClient() (*http.Client, string, error) {

	var (
		err error

		certPool *x509.CertPool
		client   *http.Client

		endpoint string
	)

	if len(i.rootCACert) > 0 {
		if certPool, err = x509.SystemCertPool(); err != nil {
			logger.DebugMessage(
				"ManagedInstance.HttpsClient(): Using new empty cert pool due to error retrieving system cert pool.: %s", 
				err.Error(),
			)
			certPool = x509.NewCertPool()
		}
		certPool.AppendCertsFromPEM([]byte(i.rootCACert))
	
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: certPool,
				},
			},
		}
	} else {
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{},
			},
		}
	}
	if endpoint, err = i.GetEndpoint(); err != nil {
		return nil, "", err
	}
	return client, endpoint, nil	
}

func (i *ManagedInstance) State() (cloud.InstanceState, error) {
	if i.Instance == nil {
		return cloud.StateUnknown, nil
	}
	return i.Instance.State()
}

func (i *ManagedInstance) CanConnect() (bool, error) {

	var (
		err  error
		port int
	)
	connError := fmt.Errorf("unable to determine connectivity state for instance")

	if len(i.hcPort) == 0 {
		logger.WarnMessage(
			"No health check will be done for managed instance '%s' as port not available.", 
			i.name,
		)
	}
	if port, err = strconv.Atoi(i.hcPort); err != nil {
		logger.ErrorMessage(
			"Invalid healthcheck port '%s' for managed instance '%s'.", 
			i.hcPort, i.name,
		)
		return false, connError
	}
	switch i.hcType {
	case "tcp":
		return i.Instance.CanConnect(port), nil
	default:
		logger.ErrorMessage(
			"Unknown health check type '%s' provided for managed instance '%s'.", 
			i.hcType, i.name,
		)
		return false, connError
	}
}
