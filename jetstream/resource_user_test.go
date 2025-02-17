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
  auth_backend = "nsc"
}

resource "jetstream_operator" "TEST" { 
  name        = "TEST"

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

 limits {
   payload            = 10000
   bearer_tokens      = true
   subscriptions      = 100
 }
 
 publish {
   allow = [ "foo.bar" ]
   deny = [ "bar.foo" ] 
 }
 
 subscribe {
   allow = [ "foo.bar" ]
   deny = [ "bar.foo" ]
 }

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
		CheckDestroy:      testUserDoesnotExist("/tmp/test/store", "TEST", "WEATHER_SERVICE", "WEATHER_USER"),
		Steps: []resource.TestStep{
			{
				Config: testUserBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("jetstream_user.WEATHER_USER", "public_key"),
					resource.TestCheckResourceAttr("jetstream_user.WEATHER_USER", "limits.0.bearer_tokens", "true"),
					resource.TestCheckResourceAttr("jetstream_user.WEATHER_USER", "limits.0.payload", "10000"),
					resource.TestCheckResourceAttr("jetstream_user.WEATHER_USER", "limits.0.subscriptions", "100"),
					resource.TestCheckResourceAttr("jetstream_user.WEATHER_USER", "publish.0.allow.0", "foo.bar"),
					resource.TestCheckResourceAttr("jetstream_user.WEATHER_USER", "publish.0.deny.0", "bar.foo"),
					resource.TestCheckResourceAttr("jetstream_user.WEATHER_USER", "subscribe.0.allow.0", "foo.bar"),
					resource.TestCheckResourceAttr("jetstream_user.WEATHER_USER", "subscribe.0.deny.0", "bar.foo"),
				),
			},
		},
	})
}
