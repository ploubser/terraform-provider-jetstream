package jetstream

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const testAccountImportBasic = `
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

resource "jetstream_account" "USERS" {
  name       = "USERS"
  operator   = "TEST"
    
  depends_on = [
    jetstream_operator.TEST
  ]  
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
  name        = "WEATHER"
  operator    = "TEST"
  account     = "WEATHER_SERVICE"
  subject     = "weather.v1.>"

  depends_on = [
    jetstream_operator.TEST,
    jetstream_account.USERS
  ]  

  // optional below
  description    = "V1 Weather Service"
  url            = "https://...."
  advertise      = true
} // should export its public key as data

resource "jetstream_account_import" "USERS_WEATHER_SERVICE" {
  name        = "USERS_WEATHER_SERVICE"
  operator    = "TEST"
  account     = "USERS"
  subject     = "weather.v1.>"
  local       = "services.weather.v1.>"
  
  source      = "WEATHER_SERVICE"

  depends_on = [
    jetstream_operator.TEST,
    jetstream_account.WEATHER_SERVICE,
    jetstream_account.USERS
  ]  

  // optional below
  share       = true
  traceable   = true
  service     = true
}
`

func TestResourceAccountImport(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: testJsProviders,
		// Write this check next
		CheckDestroy: testAccountDoesnotExist("/tmp/test/store", "TEST", "WEATHER_SERVICE"),
		Steps: []resource.TestStep{
			{
				Config: testAccountImportBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("jetstream_account_import.USERS_WEATHER_SERVICE", "id", "USERS_WEATHER_SERVICE"),
				),
			},
		},
	})
}
