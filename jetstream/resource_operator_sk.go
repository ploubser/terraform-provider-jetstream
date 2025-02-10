package jetstream

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceOperatorSk() *schema.Resource {
	return &schema.Resource{
		Create: resourceOperatorSkCreate,
		Read:   resourceOperatorSkRead,
		Delete: resourceOperatorSkDelete,
		//	Update: resourceOperatorSkUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"operator": {
				Type:         schema.TypeString,
				Description:  "The operator name",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			// Exported
			"public_key": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

func resourceOperatorSkCreate(d *schema.ResourceData, m any) error {
	conf := m.(ProviderConfig)
	auth, err := NewAuthProvider(conf)
	if err != nil {
		return err
	}

	operatorname := d.Get("operator").(string)
	operator, err := auth.Operators().Get(operatorname)
	if err != nil {
		return err
	}

	key, err := operator.SigningKeys().Add()
	if err != nil {
		return err
	}

	err = auth.Commit()
	if err != nil {
		return err
	}

	d.Set("public_key", key)
	d.SetId(key)

	return nil
}

// TODO
func resourceOperatorSkRead(d *schema.ResourceData, m any) error {
	return nil
}

func resourceOperatorSkDelete(d *schema.ResourceData, m any) error {
	conf := m.(ProviderConfig)
	auth, err := NewAuthProvider(conf)
	if err != nil {
		return err
	}

	operatorname := d.Get("operator").(string)
	operator, err := auth.Operators().Get(operatorname)
	if err != nil {
		return err
	}

	key := d.Get("public_key").(string)
	found, err := operator.SigningKeys().Delete(key)
	// TODO(ploubser): Make sure this "found" value is actaully what you expect it to be
	if !found {
		return fmt.Errorf("invalid signing key, something bad happened")
	}
	if err != nil {
		return err
	}

	err = auth.Commit()
	if err != nil {
		return err
	}

	return nil
}

// TODO
//func resourceOperatorSkUpdate(d *schema.ResourceData, m any) error {
//	return nil
//}
