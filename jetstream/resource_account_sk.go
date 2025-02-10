package jetstream

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceAccountSk() *schema.Resource {
	return &schema.Resource{
		Create: resourceAccountSkCreate,
		Read:   resourceAccountSkRead,
		Delete: resourceAccountSkDelete,
		//		Update: resourceAccountSkUpdate,
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
			"account": {
				Type:         schema.TypeString,
				Description:  "The account name",
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

func resourceAccountSkCreate(d *schema.ResourceData, m any) error {
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

	accountName := d.Get("account").(string)
	account, err := operator.Accounts().Get(accountName)
	if err != nil {
		return err
	}

	key, err := account.ScopedSigningKeys().Add()
	if err != nil {
		return err
	}

	err = auth.Commit()
	if err != nil {
		return err
	}

	// This can't stay here, but leave it here for testing
	d.Set("public_key", key)
	d.SetId(key)

	return nil
}

// TODO
func resourceAccountSkRead(d *schema.ResourceData, m any) error {
	return nil
}

// TODO
func resourceAccountSkDelete(d *schema.ResourceData, m any) error {
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

	accountName := d.Get("account").(string)
	account, err := operator.Accounts().Get(accountName)
	if err != nil {
		return err
	}

	key := d.Get("public_key").(string)
	found, err := account.ScopedSigningKeys().Delete(key)
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
//func resourceAccountSkUpdate(d *schema.ResourceData, m any) error {
//	return nil
//}
