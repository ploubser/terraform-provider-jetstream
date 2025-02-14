package jetstream

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	authb "github.com/synadia-io/jwt-auth-builder.go"
)

func resourceAccount() *schema.Resource {
	return &schema.Resource{
		Create: resourceAccountCreate,
		Read:   resourceAccountRead,
		Delete: resourceAccountDelete,
		Update: resourceAccountUpdate,
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
			"expiry": {
				Type:         schema.TypeString,
				Description:  "Sets an expiration date for the account JWT using a RFC3339 timestamp",
				Optional:     true,
				ForceNew:     false,
				ValidateFunc: validation.IsRFC3339Time,
			},
			"limits": {
				Type:        schema.TypeList,
				Description: "Limits set for the account",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bearer_tokens": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"subscriptions": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"connections": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"payload": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"leafnodes": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"imports": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"exports": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
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

	limits, set := d.GetOk("limits")

	if set {
		limitsRaw := limits.([]any)[0]
		limitsMap := limitsRaw.(map[string]any)

		bearer_tokens, is_set := limitsMap["bearer_tokens"]
		if is_set {
			account.Limits().SetDisallowBearerTokens(!bearer_tokens.(bool))
		}

		subscriptions, is_set := limitsMap["subscriptions"]
		if is_set {
			account.Limits().SetMaxSubscriptions(int64(subscriptions.(int)))
		}

		connections, is_set := limitsMap["connections"]
		if is_set {
			account.Limits().SetMaxConnections(int64(connections.(int)))
		}

		payload, is_set := limitsMap["payload"]
		if is_set {
			account.Limits().SetMaxPayload(int64(payload.(int)))
		}

		leafnodes, is_set := limitsMap["leafnodes"]
		if is_set {
			account.Limits().SetMaxLeafNodeConnections(int64(leafnodes.(int)))
		}

		imports, is_set := limitsMap["imports"]
		if is_set {
			account.Limits().SetMaxImports(int64(imports.(int)))
		}

		exports, is_set := limitsMap["exports"]
		if is_set {
			account.Limits().SetMaxExports(int64(exports.(int)))
		}
	}

	expiryString, isSet := d.GetOk("expiry")
	if isSet {
		parsedTime, err := time.Parse(time.RFC3339, expiryString.(string))
		if err != nil {
			return err
		}

		err = account.SetExpiry(parsedTime.Unix())
		if err != nil {
			return err
		}
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

	accountname := d.Get("name").(string)
	account, err := operator.Accounts().Get(accountname)
	if err != nil {
		return err
	}

	err = d.Set("expiry", time.Unix(account.Expiry(), 0).Format(time.RFC3339))
	if err != nil {
		return err
	}

	limits := []any{
		map[string]any{
			"bearer_tokens": !account.Limits().DisallowBearerTokens(),
			"connections":   account.Limits().MaxConnections(),
			"leafnodes":     account.Limits().MaxLeafNodeConnections(),
			"payload":       account.Limits().MaxPayload(),
			"subscriptions": account.Limits().MaxSubscriptions(),
			"imports":       account.Limits().MaxImports(),
			"exports":       account.Limits().MaxExports(),
		},
	}

	err = d.Set("limits", limits)
	if err != nil {
		return err
	}

	return nil
}

// HERE(ploubser): We cannot currently delete system accounts
func resourceAccountDelete(d *schema.ResourceData, m any) error {
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

	accountname := d.Get("name").(string)
	err = operator.Accounts().Delete(accountname)
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
	err = os.RemoveAll(filepath.Join(conf.StoreDirPath, operatorname, "accounts", accountname))
	if err != nil {
		fmt.Println("Error removing account directory:", err)
	}

	return nil
}

func resourceAccountUpdate(d *schema.ResourceData, m any) error {
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

	accountname := d.Get("name").(string)
	account, err := operator.Accounts().Get(accountname)
	if err != nil {
		return err
	}

	limits, set := d.GetOk("limits")

	if set {
		limitsRaw := limits.([]any)[0]
		limitsMap := limitsRaw.(map[string]any)

		bearer_tokens, is_set := limitsMap["bearer_tokens"]
		if is_set {
			account.Limits().SetDisallowBearerTokens(!bearer_tokens.(bool))
		}

		subscriptions, is_set := limitsMap["subscriptions"]
		if is_set {
			account.Limits().SetMaxSubscriptions(int64(subscriptions.(int)))
		}

		connections, is_set := limitsMap["connections"]
		if is_set {
			account.Limits().SetMaxConnections(int64(connections.(int)))
		}

		payload, is_set := limitsMap["payload"]
		if is_set {
			account.Limits().SetMaxPayload(int64(payload.(int)))
		}

		leafnodes, is_set := limitsMap["leafnodes"]
		if is_set {
			account.Limits().SetMaxLeafNodeConnections(int64(leafnodes.(int)))
		}

		imports, is_set := limitsMap["imports"]
		if is_set {
			account.Limits().SetMaxImports(int64(imports.(int)))
		}

		exports, is_set := limitsMap["exports"]
		if is_set {
			account.Limits().SetMaxExports(int64(exports.(int)))
		}
	}

	expiryString, isSet := d.GetOk("expiry")
	if isSet {
		parsedTime, err := time.Parse(time.RFC3339, expiryString.(string))
		if err != nil {
			return err
		}

		err = account.SetExpiry(parsedTime.Unix())
		if err != nil {
			return err
		}
	}

	err = auth.Commit()
	if err != nil {
		return err
	}

	return nil
}
