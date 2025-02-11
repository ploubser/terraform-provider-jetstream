package jetstream

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceOperator() *schema.Resource {
	return &schema.Resource{
		Create: resourceOperatorCreate,
		Read:   resourceOperatorRead,
		Delete: resourceOperatorDelete,
		Update: resourceOperatorUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Description:  "The operator name",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"operator_service_url": {
				Type:         schema.TypeString,
				Description:  "Operator's API endpoint for account-related operations",
				Optional:     true,
				ForceNew:     false,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"account_server_url": {
				Type:         schema.TypeString,
				Description:  "HTTP endpoint where NATS servers can dynamically fetch Account JWTs",
				Optional:     true,
				ForceNew:     false,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"tags": {
				Type:        schema.TypeList,
				Description: "Tags to group the operator",
				Optional:    true,
				ForceNew:    false,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"expiry": {
				Type:         schema.TypeString,
				Description:  "Sets an expiration date for the operator JWT, duration specified in seconds",
				Optional:     true,
				ForceNew:     false,
				ValidateFunc: validation.IsRFC3339Time,
			},
			// Exported
			"public_key": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceOperatorCreate(d *schema.ResourceData, m any) error {
	conf := m.(ProviderConfig)
	auth, err := NewAuthProvider(conf)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	o, err := auth.Operators().Add(name)
	if err != nil {
		return err
	}

	serviceUrl := d.Get("operator_service_url").(string)
	err = o.SetOperatorServiceURL(serviceUrl)
	if err != nil {
		return err
	}

	accountServer := d.Get("account_server_url").(string)
	err = o.SetAccountServerURL(accountServer)
	if err != nil {
		return err
	}

	tags := []string{}
	for _, tag := range d.Get("tags").([]any) {
		tags = append(tags, tag.(string))
	}

	err = o.Tags().Set(tags...)
	if err != nil {
		return err
	}

	expiryString := d.Get("expiry").(string)
	x, err := time.Parse(time.RFC3339, expiryString)
	if err != nil {
		return err
	}

	err = o.SetExpiry(x.Unix())
	if err != nil {
		return err
	}

	err = auth.Commit()
	if err != nil {
		return err
	}

	d.SetId(name)
	d.Set("public_key", o.JWT())

	return nil
}

func resourceOperatorRead(d *schema.ResourceData, m any) error {
	conf := m.(ProviderConfig)
	auth, err := NewAuthProvider(conf)
	if err != nil {
		return err
	}

	o, err := auth.Operators().Get(d.Id())
	if err != nil {
		return err
	}

	err = d.Set("name", o.Name())
	if err != nil {
		return err
	}

	if len(o.OperatorServiceURLs()) > 0 {
		err = d.Set("operator_service_url", o.OperatorServiceURLs()[0])
		if err != nil {
			return err
		}
	}

	err = d.Set("account_server_url", o.AccountServerURL())
	if err != nil {
		return err
	}

	tags, err := o.Tags().All()
	if err != nil {
		return err
	}

	err = d.Set("tags", tags)
	if err != nil {
		return err
	}

	err = d.Set("expiry", time.Unix(o.Expiry(), 0).Format(time.RFC3339))
	if err != nil {
		return err
	}

	return nil
}

func resourceOperatorDelete(d *schema.ResourceData, m any) error {
	conf := m.(ProviderConfig)
	auth, err := NewAuthProvider(conf)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	err = auth.Operators().Delete(name)
	if err != nil {
		return err
	}

	err = auth.Commit()
	if err != nil {
		return err
	}

	// HERE(ploubser): This is temporary while using the NSC provider, since it can't remove operators
	// Delete the operator's directory
	// We need to also delete the corresponding key but not sure how to do that without removing the other keys
	if conf.AuthBackend == "nsc" {
		err = os.RemoveAll(filepath.Join(conf.StoreDirPath, name))
		if err != nil {
			fmt.Println("Error removing operator directory:", err)
		}
	}

	return nil
}

func resourceOperatorUpdate(d *schema.ResourceData, m any) error {
	conf := m.(ProviderConfig)
	auth, err := NewAuthProvider(conf)
	if err != nil {
		return err
	}

	o, err := auth.Operators().Get(d.Id())
	if err != nil {
		return err
	}

	serviceUrl := d.Get("operator_service_url").(string)
	err = o.SetOperatorServiceURL(serviceUrl)
	if err != nil {
		return err
	}

	accountServer := d.Get("account_server_url").(string)
	err = o.SetAccountServerURL(accountServer)
	if err != nil {
		return err
	}

	tags := []string{}
	for _, tag := range d.Get("tags").([]any) {
		tags = append(tags, tag.(string))
	}

	err = o.Tags().Set(tags...)
	if err != nil {
		return err
	}

	expiryString := d.Get("expiry").(string)
	x, err := time.Parse(time.RFC3339, expiryString)
	if err != nil {
		return err
	}

	err = o.SetExpiry(x.Unix())
	if err != nil {
		return err
	}

	err = auth.Commit()
	if err != nil {
		return err
	}

	return nil
}
