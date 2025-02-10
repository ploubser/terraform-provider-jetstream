package jetstream

import (
	"fmt"
	"os"
	"path/filepath"

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
			// Not in Arri's definition but they need to be named for auth builder
			"name": {
				Type:         schema.TypeString,
				Description:  "The operator name",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"service_url": {
				Type:         schema.TypeString,
				Description:  "",
				Optional:     true,
				ForceNew:     false,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"tags": {
				Type:        schema.TypeList,
				Description: "",
				Optional:    true,
				ForceNew:    false,
				Elem:        &schema.Schema{Type: schema.TypeString},
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

	err = auth.Commit()
	if err != nil {
		return err
	}

	d.SetId(name)
	d.Set("public_key", o.JWT())

	return nil
}

// TODO
func resourceOperatorRead(d *schema.ResourceData, m any) error {
	return nil
}

func resourceOperatorDelete(d *schema.ResourceData, m any) error {
	conf := m.(ProviderConfig)
	auth, err := NewAuthProvider(conf)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	// This is suppose to do something?
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
	// Are there any other files that need to be deleted?
	err = os.RemoveAll(filepath.Join(conf.StoreDirPath, name))
	if err != nil {
		fmt.Println("Error removing operator directory:", err)
	}

	return nil
}

// TODO
func resourceOperatorUpdate(d *schema.ResourceData, m any) error {
	return nil
}
