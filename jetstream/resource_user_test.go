package jetstream

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/nats-io/jsm.go"
	"github.com/nats-io/nats.go"
)

const testUserBasic = `
provider "jetstream" {
  servers = "foo"
  store_dir_path = "/tmp/test/store"
  keys_dir_path = "/tmp/test/keys"
  auth_provider = "nsc"
}

resource "jetstream_operator" "TEST" { 
  name        = "TEST"
  service_url = "https://...." // optional
  tags        = ["foo", "bar"]     // optional

} 
resource "jetstream_account_sk" "FOO" { 
  operator   = "TEST"
  account = "WEATHER_SERVICE"
  depends_on = [
    jetstream_operator.TEST,
	jetstream_account.WEATHER_SERVICE
  ]
} 

resource "jetstream_account" "WEATHER_SERVICE" {
  name       = "WEATHER_SERVICE"
  operator   = "TEST"

  depends_on = [
    jetstream_operator.TEST,
  ]  
} 

resource "jetstream_user" "WEATHER_USER" {
 name        = "WEATHER_USER"
 operator    = "TEST"
 account     = "WEATHER_SERVICE"
 signing_key = jetstream_account_sk.FOO.public_key

 depends_on = [
    jetstream_operator.TEST,
    jetstream_account.WEATHER_SERVICE,
	jetstream_account_sk.FOO
  ]
}
`

func TestResourceUser(t *testing.T) {
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
		//CheckDestroy:      testOperatorDoesnotExist("/tmp/test/store", "TEST2"),
		Steps: []resource.TestStep{
			{
				Config: testUserBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("jetstream_user.WEATHER_USER", "public_key"),
					// Check jwt contents and see if it matches what we expect
				),
			},
		},
	})
}
