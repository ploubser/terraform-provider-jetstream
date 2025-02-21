package jetstream

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const testAccountExportBasic = `
provider "jetstream" {
  servers = "foo"
  store_dir_path = "/tmp/test/store"
  keys_dir_path = "/tmp/test/keys"
  auth_backend = "nsc"
}

resource "jetstream_operator" "TEST" { 
  name        = "TEST"
  operator_service_url = "nats://localhost:4222"
  account_server_url = "https://jwt-resolver.example.com"
  tags        = ["foo", "bar"]    
  expiry      = "2100-02-11T00:00:00Z"

} 
resource "jetstream_account" "WEATHER_SERVICE" {
  name       = "WEATHER_SERVICE"
  operator   = "TEST"

  limits  {                                  
    bearer_tokens = true
    connections   = 1000
    leafnodes     = 100
    payload       = 1024
    subscriptions = 100000
	imports       = 91
	exports       = 19
  }

  tags        = ["foo", "bar"]    
  expiry = "2100-02-11T00:00:00Z"

  depends_on = [
    jetstream_operator.TEST
  ]  
}

resource "jetstream_account_export" "WEATHER" {
  name        = "weather_export"
  operator    = "TEST"
  account     = "WEATHER_SERVICE"
  subject     = "weather.v1.*.>"

  depends_on = [
    jetstream_operator.TEST,
    jetstream_account.WEATHER_SERVICE
  ]  

  // optional below
  description    = "V1 Weather Service"
  url            = "https://...."
  token_position = 3
  advertise      = true
} // should export its public key as data
`

func TestResourceAccountExport(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: testJsProviders,
		// Write this check next
		//CheckDestroy: testAccountDoesnotExist("/tmp/test/store", "TEST", "WEATHER"),
		Steps: []resource.TestStep{
			{
				Config: testAccountExportBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("jetstream_account_export.WEATHER", "id", "weather_export"),
				),
			},
		},
	})
}
