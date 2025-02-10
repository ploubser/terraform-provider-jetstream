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
  service_url = "https://...." // optional
  tags        = ["foo", "bar"]     // optional

} 

resource "jetstream_account" "WEATHER_SERVICE" {
  name       = "WEATHER_SERVICE"
  operator   = "TEST"

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
		//CheckDestroy:      testOperatorDoesnotExist("/tmp/test/store", "TEST2"),
		Steps: []resource.TestStep{
			{
				Config: testAccountBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("jetstream_account.WEATHER_SERVICE", "public_key"),
					// Check jwt contents and see if it matches what we expect
				),
			},
		},
	})
}
