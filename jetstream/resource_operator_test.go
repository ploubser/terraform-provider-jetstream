package jetstream

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/nats-io/jsm.go"
	"github.com/nats-io/nats.go"
)

const testOperatorBasic = `
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
`

func TestResourceOperator(t *testing.T) {
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
		CheckDestroy:      testOperatorDoesnotExist("/tmp/test/store", "TEST"),
		Steps: []resource.TestStep{
			{
				Config: testOperatorBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("jetstream_operator.TEST", "public_key"),
					resource.TestCheckResourceAttr("jetstream_operator.TEST", "operator_service_url", "nats://localhost:4222"),
					resource.TestCheckResourceAttr("jetstream_operator.TEST", "account_server_url", "https://jwt-resolver.example.com"),
					resource.TestCheckResourceAttr("jetstream_operator.TEST", "tags.0", "foo"),
					resource.TestCheckResourceAttr("jetstream_operator.TEST", "tags.1", "bar"),
					resource.TestCheckResourceAttr("jetstream_operator.TEST", "expiry", "2100-02-11T00:00:00Z"),
				),
			},
		},
	})
}
