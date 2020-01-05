module github.com/appbricks/cloud-builder

go 1.13

replace github.com/appbricks/cloud-builder => ./

replace github.com/mevansam/gocloud => ../../mevansam/gocloud

replace github.com/mevansam/goforms => ../../mevansam/goforms

replace github.com/mevansam/goutils => ../../mevansam/goutils

require (
	github.com/Azure/azure-sdk-for-go v37.2.0+incompatible
	github.com/Azure/go-autorest/autorest v0.9.3
	github.com/Azure/go-autorest/autorest/adal v0.8.1
	github.com/aws/aws-lambda-go v1.13.3
	github.com/aws/aws-sdk-go v1.27.0
	github.com/gobuffalo/packr/v2 v2.7.1
	github.com/hashicorp/hcl/v2 v2.2.0
	github.com/hashicorp/terraform v0.12.18
	github.com/kr/pretty v0.2.0
	github.com/mevansam/gocloud v0.0.0-00010101000000-000000000000
	github.com/mevansam/goforms v0.0.0-00010101000000-000000000000
	github.com/mevansam/goutils v0.0.0-00010101000000-000000000000
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/spf13/viper v1.6.1
	github.com/zclconf/go-cty v1.1.1
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	google.golang.org/api v0.15.0
)
