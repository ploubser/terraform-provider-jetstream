package jetstream

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceUser() *schema.Resource {
	return &schema.Resource{
		Create: resourceUserCreate,
		Read:   resourceUserRead,
		Delete: resourceUserDelete,
		Update: resourceUserUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Description:  "The user name",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
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
			"signing_key": {
				Type:         schema.TypeString,
				Description:  "The signing key",
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

func resourceUserCreate(d *schema.ResourceData, m any) error {
	conf := m.(ProviderConfig)
	auth, err := NewAuthProvider(conf)
	if err != nil {
		return err
	}

	opname := d.Get("operator").(string)
	o, err := auth.Operators().Get(opname)
	if err != nil {
		return err
	}

	accountname := d.Get("account").(string)
	account, err := o.Accounts().Get(accountname)
	if err != nil {
		return fmt.Errorf("cannot find account '%s'. Error: '%s'", accountname, err)
	}

	username := d.Get("name").(string)
	signer := d.Get("signing_key").(string)
	user, err := account.Users().Add(username, signer)
	if err != nil {
		return err
	}

	// do things with user

	err = auth.Commit()
	if err != nil {
		return err
	}

	d.Set("public_key", user.JWT())
	d.SetId(username)

	return nil
}

func resourceUserRead(d *schema.ResourceData, m any) error {
	return nil
}

func resourceUserDelete(d *schema.ResourceData, m any) error {
	conf := m.(ProviderConfig)
	auth, err := NewAuthProvider(conf)
	if err != nil {
		return err
	}

	opname := d.Get("operator").(string)
	o, err := auth.Operators().Get(opname)
	if err != nil {
		return err
	}

	accountname := d.Get("account").(string)
	account, err := o.Accounts().Get(accountname)
	if err != nil {
		return err
	}

	username := d.Get("name").(string)
	account.Users().Delete(username)
	return nil
}

func resourceUserUpdate(d *schema.ResourceData, m any) error {
	return nil
}
