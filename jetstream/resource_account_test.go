package jetstream

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/nats-io/jsm.go"
	"github.com/nats-io/nats.go"
)

const testAccountBasic = `
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

// TODO(ploubser):
// Test system account when we can delete it
// Test signing keys

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
`

func TestResourceAccount(t *testing.T) {
	srv := createJSServer(t)
	defer srv.Shutdown()

	nc, err := nats.Connect(srv.ClientURL())
	if err != nil {
		t.Fatalf("could not connect: %s", err)
	}
	defer nc.Close()

	_, err = jsm.New(nc)
	if err != nil {
		t.Fatalf("could not connect: %s", err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: testJsProviders,
		// Write this check next
		CheckDestroy: testAccountDoesnotExist("/tmp/test/store", "TEST", "WEATHER_SERVICE"),
		Steps: []resource.TestStep{
			{
				Config: testAccountBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("jetstream_account.WEATHER_SERVICE", "public_key"),
					resource.TestCheckResourceAttr("jetstream_account.WEATHER_SERVICE", "tags.0", "foo"),
					resource.TestCheckResourceAttr("jetstream_account.WEATHER_SERVICE", "tags.1", "bar"),
					resource.TestCheckResourceAttr("jetstream_account.WEATHER_SERVICE", "limits.0.bearer_tokens", "true"),
					resource.TestCheckResourceAttr("jetstream_account.WEATHER_SERVICE", "limits.0.connections", "1000"),
					resource.TestCheckResourceAttr("jetstream_account.WEATHER_SERVICE", "limits.0.leafnodes", "100"),
					resource.TestCheckResourceAttr("jetstream_account.WEATHER_SERVICE", "limits.0.payload", "1024"),
					resource.TestCheckResourceAttr("jetstream_account.WEATHER_SERVICE", "limits.0.subscriptions", "100000"),
					resource.TestCheckResourceAttr("jetstream_account.WEATHER_SERVICE", "limits.0.imports", "91"),
					resource.TestCheckResourceAttr("jetstream_account.WEATHER_SERVICE", "limits.0.exports", "19"),
				),
			},
		},
	})
}
