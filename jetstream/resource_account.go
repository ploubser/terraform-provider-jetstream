package jetstream

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	authb "github.com/synadia-io/jwt-auth-builder.go"
)

func resourceAccount() *schema.Resource {
	return &schema.Resource{
		Create: resourceAccountCreate,
		Read:   resourceAccountRead,
		Delete: resourceAccountDelete,
		//Update: resourceAccountUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Description:  "The account name",
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
			"system": {
				Type:        schema.TypeBool,
				Description: "Is this a system account",
				Optional:    true,
				ForceNew:    true,
			},
			"operator_signing_key": {
				Type:         schema.TypeString,
				Description:  "The operator's signing key",
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			// Exported
			"public_key": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAccountCreate(d *schema.ResourceData, m any) error {
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

	accountName := d.Get("name").(string)
	var account authb.Account
	operatorSigningKey := d.Get("operator_signing_key").(string)

	if operatorSigningKey != "" {
		account, err = authb.NewAccountFromJWT(operatorSigningKey)
		if err != nil {
			return err
		}
	} else {
		account, err = operator.Accounts().Add(accountName)
		if err != nil {
			return err
		}
	}

	if d.Get("system").(bool) {
		operator.SetSystemAccount(account)
	}

	err = auth.Commit()
	if err != nil {
		return err
	}

	d.SetId(accountName)
	d.Set("public_key", account.JWT())

	return nil
}

func resourceAccountRead(d *schema.ResourceData, m any) error {
	return nil
}

// HERE(ploubser): We cannot currently delete system accounts
func resourceAccountDelete(d *schema.ResourceData, m any) error {
	/*	conf := m.(ProviderConfig)
		auth, err := NewAuthProvider(conf)
		if err != nil {
			return err
		}

		operatorname := d.Get("operator").(string)
		operator, err := auth.Operators().Get(operatorname)
		if err != nil {
			return err
		}

		accountName := d.Get("name").(string)
		err = operator.Accounts().Delete(accountName)
		if err != nil {
			return err
		}

		err = auth.Commit()
		if err != nil {
			return err
		}

		// HERE(ploubser): This is temporary while using the NSC provider, since it can't remove accounts
		// Delete the accounts' directory
		// Are there any other files that need to be deleted?
		err = os.RemoveAll(filepath.Join(conf.StoreDirPath, operatorname, "accounts", accountName))
		if err != nil {
			fmt.Println("Error removing account directory:", err)
		}
	*/
	return nil
}

//func resourceAccountUpdate(d *schema.ResourceData, m any) error {
//	return nil
//}
