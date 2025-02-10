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
  service_url = "https://...." // optional
  tags        = ["foo", "bar"]     // optional
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
		CheckDestroy:      testOperatorDoesnotExist("/tmp/test/store", "TEST2"),
		Steps: []resource.TestStep{
			{
				Config: testOperatorBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("jetstream_operator.TEST", "public_key"),
					resource.TestCheckResourceAttr("jetstream_operator.TEST", "service_url", "https://...."),
					resource.TestCheckResourceAttr("jetstream_operator.TEST", "tags.0", "foo"),
					resource.TestCheckResourceAttr("jetstream_operator.TEST", "tags.1", "bar"),
					// Check jwt contents and see if it matches what we expect
				),
			},
		},
	})
}
