package jetstream

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const testAccountSkBasic = `
provider "jetstream" {
  servers = "foo"
  auth_backend = "nsc"
  store_dir_path = "/tmp/test/store"
  keys_dir_path = "/tmp/test/keys"
}

resource "jetstream_operator" "TEST" { 
  name        = "TEST"
} 


resource "jetstream_account" "WEATHER_SERVICE" {
  name       = "WEATHER_SERVICE"
  operator   =  "TEST"

  depends_on = [
    jetstream_operator.TEST,
  ]  
} 

resource "jetstream_account_sk" "UNSCOPED" { 
  operator   = "TEST"
  account = "WEATHER_SERVICE"

  depends_on = [
    jetstream_account.WEATHER_SERVICE
  ]  
} 
`

const testAccountSkScoped = `
provider "jetstream" {
  servers = "foo"
  auth_backend = "nsc"
  store_dir_path = "/tmp/test/store"
  keys_dir_path = "/tmp/test/keys"
}

resource "jetstream_operator" "TEST" { 
  name        = "TEST"
} 


resource "jetstream_account" "WEATHER_SERVICE" {
  name       = "WEATHER_SERVICE"
  operator   =  jetstream_operator.TEST.name

  depends_on = [
    jetstream_operator.TEST,
  ]  
} 

resource "jetstream_account_sk" "SCOPED" { 
  operator   = jetstream_operator.TEST.name
  account = "WEATHER_SERVICE"
  role  = "test_role"

  limits {
	  bearer_tokens = false
  }

  depends_on = [
    jetstream_operator.TEST,
	  jetstream_account.WEATHER_SERVICE
  ]
} 
`

func TestResourceAccountSK(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: testJsProviders,
		// TODO(ploubser): Rewrite these tests
		// CheckDestroy:      testAccountSigningKeyDoesnotExist("/tmp/test/store", "/tmp/test/key", "TEST", "WEATHER_SERVICE"),
		Steps: []resource.TestStep{
			{
				Config: testAccountSkBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("jetstream_account_sk.UNSCOPED", "public_key"),
				),
			},
			{
				Config: testAccountSkScoped,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("jetstream_account_sk.SCOPED", "public_key"),
				),
			},
		},
	})
}
