package mocks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/appbricks/cloud-builder/cookbook"
	"github.com/appbricks/cloud-builder/target"
	"github.com/appbricks/cloud-builder/terraform"
	"github.com/mevansam/gocloud/backend"
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goutils/run"

	backend_mocks "github.com/mevansam/gocloud/test/mocks"
	provider_mocks "github.com/mevansam/gocloud/test/mocks"
)

type FakeTargetContext struct {
	recipePath string

	targets *target.TargetSet
}

func NewTargetMockContext(
	recipePath string,
) *FakeTargetContext {

	return &FakeTargetContext{
		recipePath: recipePath,
		targets: target.NewTargetSet(nil),
	}
}
	
func (mctx *FakeTargetContext) Reset() error {
	return nil
}

func (mctx *FakeTargetContext) Load(input io.Reader) error {
	return nil
}
	
func (mctx *FakeTargetContext) Save(output io.Writer) error {
	return nil
}

func (mctx *FakeTargetContext) Cookbook() *cookbook.Cookbook {
	return nil
}
	
func (mctx *FakeTargetContext) GetCookbookRecipe(recipe, iaas string) (cookbook.Recipe, error) {
	return nil, nil
}
	
func (mctx *FakeTargetContext) SaveCookbookRecipe(recipe cookbook.Recipe) {
}

func (mctx *FakeTargetContext) CloudProviderTemplates() []provider.CloudProvider {
	return nil
}
	
func (mctx *FakeTargetContext) GetCloudProvider(iaas string) (provider.CloudProvider, error) {
	return nil, nil
}
	
func (mctx *FakeTargetContext) SaveCloudProvider(provider provider.CloudProvider) {
}

func (mctx *FakeTargetContext) NewTarget(recipeKey, recipeIaas string,) (*target.Target, error) {

	var (
		err error

		r cookbook.Recipe
		p provider.CloudProvider
		b backend.CloudBackend
	)

	recipeElems := strings.Split(recipeKey, ":")
	cookbookName := recipeElems[0]
	recipeName := recipeElems[1]

	if r, err = cookbook.NewRecipe(
		recipeKey, recipeIaas,
		fmt.Sprintf("%s/%s/%s", mctx.recipePath, recipeName, recipeIaas),
		"", "", "", "", "", cookbookName, "0.0.0", recipeName, [][]string{}); err != nil {
		return nil, err
	}
	if p, err = provider.NewCloudProvider(recipeIaas); err != nil {
		return nil, err
	}
	if b, err = backend.NewCloudBackend(r.BackendType()); err != nil {
		return nil, err
	}
	return target.NewTarget(r, p, b), nil
}

func (mctx *FakeTargetContext) TargetSet() *target.TargetSet {
	return mctx.targets
}
	
func (mctx *FakeTargetContext) HasTarget(name string) bool {
	return false
}
	
func (mctx *FakeTargetContext) GetTarget(name string) (*target.Target, error) {
	return mctx.targets.GetTarget(name), nil
}
	
func (mctx *FakeTargetContext) SaveTarget(key string, target *target.Target) {
	err := mctx.targets.SaveTarget(key, target)
	if err != nil {
		panic(err)
	}
}

func (mctx *FakeTargetContext) DeleteTarget(key string) {
	mctx.targets.DeleteTarget(key)
}

func (mctx *FakeTargetContext) IsDirty() bool {
	return false
}

func NewMockTarget(cli run.CLI, bastionIP string, bastionPort int, caRootPEM string) *target.Target {

	var (
		err error

		tmpl       *template.Template
		tmplResult bytes.Buffer
	)

	tmplVars := struct {
		BastionIP   string
		BastionPort int
		CARootPEM   string
	}{
		BastionIP: bastionIP,
		BastionPort: bastionPort,
		CARootPEM: strings.Replace(caRootPEM, "\n", "\\n", -1),
	}

	if tmpl, err = template.New("terraformOutput").Parse(terraformOutput); err != nil {
		panic(err)
	}
	if err = tmpl.Execute(&tmplResult, tmplVars); err != nil {
		panic(err)
	}

	output := make(map[string]terraform.Output)
	if err = json.Unmarshal(tmplResult.Bytes(), &output); err != nil {
		panic(err)
	}

	recipe := NewFakeRecipe(cli)
	recipe.SetBastion()

	return &target.Target{
		RecipeName: "fakeRecipe",
		RecipeIaas: "fakeIAAS",

		Recipe: recipe,
		Provider: provider_mocks.NewFakeCloudProvider(),
		Backend: backend_mocks.NewFakeCloudBackend(),

		Output: &output,

		RSAPrivateKey: targetRSAPrivateKey,
		RSAPublicKey: targetRSAPublicKey,

		NodeKey: targetSpaceKey,
		NodeID: targetSpaceID,
	}
}

// mock data

const terraformOutput = `{
	"cb_default_openssh_private_key": {
		"Sensitive": false,
		"Type": "string",
		"Value": ""
	},
	"cb_default_ssh_key_pair": {
		"Sensitive": false,
		"Type": "string",
		"Value": "mycs-dev-test-us-east-1"
	},
	"cb_deployment_networks": {
		"Sensitive": false,
		"Type": [
			"tuple",
			[
				"string"
			]
		],
		"Value": [
			"subnet-0302259f81b9a4d59"
		]
	},
	"cb_deployment_security_group": {
		"Sensitive": false,
		"Type": "string",
		"Value": "sg-099b7bd3c3b633b4c"
	},
	"cb_dns_configured": {
		"Sensitive": false,
		"Type": "bool",
		"Value": false
	},
	"cb_idle_action": {
		"Sensitive": false,
		"Type": "string",
		"Value": "shutdown"
	},
	"cb_internal_pdns_api_key": {
		"Sensitive": false,
		"Type": "string",
		"Value": "8W?+*(oeL=3l(qK#!xqaAh3u{d@waouE"
	},
	"cb_internal_pdns_url": {
		"Sensitive": false,
		"Type": "string",
		"Value": "http://10.0.16.253:8888"
	},
	"cb_managed_instances": {
		"Sensitive": false,
		"Type": [
			"tuple",
			[
				[
					"object",
					{
						"api_port": "string",
						"description": "string",
						"fqdn": "string",
						"id": "string",
						"name": "string",
						"non_root_passwd": "string",
						"non_root_user": "string",
						"order": "number",
						"private_ip": "string",
						"public_ip": "string",
						"root_passwd": "string",
						"root_user": "string",
						"ssh_key": "string",
						"ssh_port": "string",
						"ssh_user": "string"
					}
				]
			]
		],
		"Value": [
			{
				"api_port": "{{ .BastionPort }}",
				"description": "The Bastion instance runs the VPN service that can be used to\nsecurely and anonymously access your cloud space resources and the\ninternet. You can download the VPN configuration along with the VPN\nclient software from the password protected links below. The same\nuser and password used to access the link should be used to the login\nto the VPN if required.\n\n* URL: https://54.158.84.168/~mycs-user\n  User: mycs-user\n  Password: @ppBr!cks2O2I\n",
				"fqdn": "",
				"id": "bastion-instance-id",
				"name": "bastion",
				"non_root_passwd": "@ppBr!cks2O2I",
				"non_root_user": "mycs-user",
				"order": 0,
				"private_ip": "{{ .BastionIP }}",
				"public_ip": "{{ .BastionIP }}",
				"root_passwd": "@ppBr!cks2O2I",
				"root_user": "mycs-admin",
				"ssh_key": "-----BEGIN RSA PRIVATE KEY-----\n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\n-----END RSA PRIVATE KEY-----\n",
				"ssh_port": "22",
				"ssh_user": "mycs-admin"
			}
		]
	},
	"cb_node_description": {
		"Sensitive": false,
		"Type": "string",
		"Value": "This My Cloud Space sandbox has been deployed to the following public\ncloud environment. Along with a sandboxed virtual cloud network it\nincludes a VPN service which allows you to access the internet as\nwell as your personal cloud space services securely while maintaining\nyour privacy.\n\nProvider: Amazon Web Services\nRegion: us-east-1\nVPN Type: WireGuard\nVersion: 0.0.3\n"
	},
	"cb_node_version": {
		"Sensitive": false,
		"Type": "string",
		"Value": "0.0.3"
	},
	"cb_root_ca_cert": {
		"Sensitive": false,
		"Type": "string",
		"Value": "{{ .CARootPEM }}"
	},
	"cb_vpc_id": {
		"Sensitive": false,
		"Type": "string",
		"Value": "vpc-0faffc9d9ce084074"
	},
	"cb_vpc_name": {
		"Sensitive": false,
		"Type": "string",
		"Value": "mycs-dev-test-us-east-1"
	},
	"cb_vpn_masking_available": {
		"Sensitive": false,
		"Type": "string",
		"Value": "no"
	},
	"cb_vpn_type": {
		"Sensitive": false,
		"Type": "string",
		"Value": "wireguard"
	}
}`

const targetRSAPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
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
IOLgmdDwWCjl8z0bZde6JLCl0Ws/
-----END RSA PRIVATE KEY-----`

const targetRSAPublicKey = `-----BEGIN PUBLIC KEY-----
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAr0LUV04g6H9lYn54O7ER
AQTUAeJouZAngiq7R/FyJnUqDWrBO5kQNamAF8vswv/3SrmfRB9DOotxYrVdz6j6
5PQKE1GKqKXIUM964iA4R+6p8S+mzW33vwDOC58rVnfISb2i5ja2R3Idx+MMbgK8
VFA7bRuj/2xo3nW2w6qHKW1isSoDTbHfTWgoJsZBMCcs5XP1RrsT9yN1GZvGiHeA
yy71j7ZRZAA8xt8CPIRvQ9X9zMj/2CAHgTTCHiFZ+O/6Nmr3A26U0IJjY7SzEjM1
unV2Ms7Snb8AHGP6WErdYlSVwk0/h4YB26CgFvAzkC4Wr1QbOhMNwmqIb0i8PPGJ
PAB6fVpM4EVZQOZMF4cYg+ZSq5vuWrrFYbsy4hW4RLgBsHrFCIZjKw2E/52RDMDB
2qOFtwniW+VaA+s22Jy1ZMwR88jcniDWA8dXfYQ+e5Nkd/hrmBa0pYLOTTQzzzhh
Mo8iTw+QxHGeHCI/Wne0n9BGk3A+H/exDVSUtCf9yGXTwlEjNqtdA09+p1OINSlW
1IVIs5or5SgLFChmuyu/DaAp7A4BywxfJ6QOaSCN8cEZI6cAS9wytCoWaStFYmPk
0tFUHEyKLDZQs/InfAnQ+G3rmQ0oJHwuYHn6x5hP4YNz285247FubfJgfU8ne8OS
IikEW5txj0gJy0yQTktTMHUCAwEAAQ==
-----END PUBLIC KEY-----`

const targetSpaceKey = `b1f187f2-1019-4848-ae7c-4db0cec1f256|mOkqjXKklRBufBOMiDRd8rH76FY8OOT/yxkxk/QUJkNn6MBUD9INzUJ+k6fcNOBsulg2EeGRjMTJ7qN6SoI6NnyQmaM9H67FrNQOcThv/yowEmU7M9KvapVYZhuBcAogRkHlA3DFOL2glN+tTwufoIifEXGoagLn7+XWtghfQeHmRmr9j/Swr/4YzCgsu5kK3wY6eKfsniDKnvi2DZcZQzQMkrPNEN+QoRrQdV8QhEFuk1eDrSfhTT++YQQvdSRQya4HPKj2U6IKV41+UCcFWIz1pjiedExjwv15IvLsuvNpqekqg63Uk7mTHbfn99CLTivwQt27lVpcxH1fG8IDCGSnSu6aTF9FzvUgatWTxPw8atzBQrd2lHYXyOU4S8U225dsWB4D+8KId3ZlXq+05vmFQqeJ1xCvgxACGM2LSRUShmhAADKjfuppRB5aKS+7vRqT1/D65yKx/N1SqhaFHZjL0Mwe58MG56yv3BkWYO1MoDt+RD1RApxvGJ3TnaS+EORMrgGCHd/5kWIRXTOSjmBV6Q4B6TCMBCNB1UOTQEWqQ02OO2xYlelAsgdOitUUS4HCUBiwRU4euwnHL9Bg5L0lshtwafVf3VmZDDbcDAhgG1fFEMFAaFl63HexSqpadNk5l9X8ai9MKzb5jl5rP8oKH8ujHUL/yWY6Zg59NnI=`
const targetSpaceID = `5ed8679e-d684-4d54-9b4f-2e73f7f8d342`
