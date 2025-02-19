package jetstream

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/nats-io/jsm.go"
	"github.com/nats-io/nats.go"
)

const testOperatorSkBasic = `
provider "jetstream" {
  servers = "foo"
  store_dir_path = "/tmp/test/store"
  keys_dir_path = "/tmp/test/keys"
  auth_backend = "nsc"
}

resource "jetstream_operator" "TEST" { 
  name        = "TEST"
} 

resource "jetstream_operator_sk" "FOO" { 
  operator   = "TEST"

  depends_on = [
    jetstream_operator.TEST
  ]
} 
`

func TestResourceOperatorSK(t *testing.T) {
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
		CheckDestroy:      testOperatorSigningKeyDoesnotExist("/tmp/test/store", "/tmp/test/store/key", "TEST"),
		Steps: []resource.TestStep{
			{
				Config: testOperatorSkBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("jetstream_operator_sk.FOO", "public_key"),
					// Check jwt contents and see if it matches what we expect
				),
			},
		},
	})
}
