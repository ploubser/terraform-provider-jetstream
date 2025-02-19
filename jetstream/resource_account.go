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
				Description: "Creates a system account",
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
				ForceNew:     true,
				ValidateFunc: validation.IsRFC3339Time,
			},
			"tags": {
				Type:        schema.TypeList,
				Description: "Tags to group the operator",
				Optional:    true,
				ForceNew:    false,
				Elem:        &schema.Schema{Type: schema.TypeString},
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
		return fmt.Errorf("unable to create auth provider: %s", err)
	}

	operatorname := d.Get("operator").(string)
	operator, err := auth.Operators().Get(operatorname)
	if err != nil {
		return fmt.Errorf("unable to load operator '%s': %s", operatorname, err)
	}

	accountname := d.Get("name").(string)
	var account authb.Account
	operatorSigningKey := d.Get("operator_signing_key").(string)

	// TODO(ploubser): Update this after operator_sk has been updated
	if operatorSigningKey != "" {
		account, err = authb.NewAccountFromJWT(operatorSigningKey)
		if err != nil {
			return fmt.Errorf("THIS FAILURE HAS A TODO")
		}
	} else {
		account, err = operator.Accounts().Add(accountname)
		if err != nil {
			return fmt.Errorf("failed to create account: %s", err)
		}
	}

	if d.Get("system").(bool) {
		operator.SetSystemAccount(account)
	}

	tags := []string{}
	if rawTags, ok := d.GetOk("tags"); ok {
		for _, tag := range rawTags.([]any) {
			tags = append(tags, tag.(string))
		}
	}

	err = account.Tags().Set(tags...)
	if err != nil {
		return fmt.Errorf("unable to create tags for account '%s': %s", accountname, err)
	}

	if limits, set := d.GetOk("limits"); set {
		if limitsMap, ok := limits.([]any)[0].(map[string]any); ok {
			mappings := map[string]func(int64) error{
				"subscriptions": account.Limits().SetMaxSubscriptions,
				"connections":   account.Limits().SetMaxConnections,
				"payload":       account.Limits().SetMaxPayload,
				"leafnodes":     account.Limits().SetMaxLeafNodeConnections,
				"imports":       account.Limits().SetMaxImports,
				"exports":       account.Limits().SetMaxExports,
			}

			if bearerTokens, ok := limitsMap["bearer_tokens"]; ok {
				account.Limits().SetDisallowBearerTokens(!bearerTokens.(bool))
			}

			for k, fn := range mappings {
				if value, ok := limitsMap[k]; ok {
					fn(int64(value.(int)))
				}
			}
		}
	}

	expiry, isSet := d.GetOk("expiry")
	if isSet {
		parsedTime, err := time.Parse(time.RFC3339, expiry.(string))
		if err != nil {
			return fmt.Errorf("unable to parse time string '%s': %s", expiry.(string), err)
		}

		err = account.SetExpiry(parsedTime.Unix())
		if err != nil {
			return fmt.Errorf("unable to set expiry for account '%s': %s", accountname, err)
		}
	}

	err = auth.Commit()
	if err != nil {
		return fmt.Errorf("unable to create account '%s': %s", accountname, err)
	}

	d.SetId(accountname)
	d.Set("public_key", account.JWT())

	return nil
}

func resourceAccountRead(d *schema.ResourceData, m any) error {
	conf := m.(ProviderConfig)
	auth, err := NewAuthProvider(conf)
	if err != nil {
		return fmt.Errorf("unable to create auth provider: %s", err)
	}

	operatorname := d.Get("operator").(string)
	operator, err := auth.Operators().Get(operatorname)
	if err != nil {
		return fmt.Errorf("unable to load operator '%s': %s", operatorname, err)
	}

	accountname := d.Get("name").(string)
	account, err := operator.Accounts().Get(accountname)
	if err != nil {
		return fmt.Errorf("failed to load account: %s", err)
	}

	expiry := account.Expiry()
	if expiry != 0 {
		err = d.Set("expiry", time.Unix(account.Expiry(), 0).Format(time.RFC3339))
		if err != nil {
			return fmt.Errorf("unable to get expiry for account '%s': %s", accountname, err)
		}
	}

	tags, err := account.Tags().All()
	if err != nil {
		return fmt.Errorf("unable to get tags for account '%s': %s", accountname, err)
	}

	err = d.Set("tags", tags)
	if err != nil {
		return fmt.Errorf("unable to set tags for account '%s': %s", accountname, err)
	}

	limits := accountLimits(account)
	if configuredLimits, set := d.GetOk("limits"); set {
		if limitsMap, ok := configuredLimits.([]any)[0].(map[string]any); ok {
			for k := range limitsMap {
				limitsMap[k] = limits[k]
			}
			err = d.Set("limits", []any{limitsMap})
			if err != nil {
				return fmt.Errorf("unable to set limits for account '%s': %s", accountname, err)
			}
		}
	}

	return nil
}

func accountLimits(account authb.Account) map[string]any {
	return map[string]any{
		"bearer_tokens": !account.Limits().DisallowBearerTokens(),
		"connections":   account.Limits().MaxConnections(),
		"leafnodes":     account.Limits().MaxLeafNodeConnections(),
		"payload":       account.Limits().MaxPayload(),
		"subscriptions": account.Limits().MaxSubscriptions(),
		"imports":       account.Limits().MaxImports(),
		"exports":       account.Limits().MaxExports(),
	}
}

// HERE(ploubser): We cannot currently delete system accounts
func resourceAccountDelete(d *schema.ResourceData, m any) error {
	conf := m.(ProviderConfig)
	auth, err := NewAuthProvider(conf)
	if err != nil {
		return fmt.Errorf("unable to create auth provider: %s", err)
	}

	operatorname := d.Get("operator").(string)
	operator, err := auth.Operators().Get(operatorname)
	if err != nil {
		return fmt.Errorf("unable to load operator '%s': %s", operatorname, err)
	}

	accountname := d.Get("name").(string)
	err = operator.Accounts().Delete(accountname)
	if err != nil {
		return fmt.Errorf("failed to load account: %s", err)
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
		return fmt.Errorf("unable to create auth provider: %s", err)
	}

	operatorname := d.Get("operator").(string)
	operator, err := auth.Operators().Get(operatorname)
	if err != nil {
		return fmt.Errorf("unable to load operator '%s': %s", operatorname, err)
	}

	accountname := d.Get("name").(string)
	account, err := operator.Accounts().Get(accountname)
	if err != nil {
		return fmt.Errorf("failed to load account: %s", err)
	}

	if limits, set := d.GetOk("limits"); set {
		if limitsMap, ok := limits.([]any)[0].(map[string]any); ok {
			mappings := map[string]func(int64) error{
				"subscriptions": account.Limits().SetMaxSubscriptions,
				"connections":   account.Limits().SetMaxConnections,
				"payload":       account.Limits().SetMaxPayload,
				"leafnodes":     account.Limits().SetMaxLeafNodeConnections,
				"imports":       account.Limits().SetMaxImports,
				"exports":       account.Limits().SetMaxExports,
			}

			if bearerTokens, ok := limitsMap["bearer_tokens"]; ok {
				account.Limits().SetDisallowBearerTokens(!bearerTokens.(bool))
			}

			for k, fn := range mappings {
				if value, ok := limitsMap[k]; ok {
					fn(int64(value.(int)))
				}
			}
		}
	}

	tags := []string{}
	for _, tag := range d.Get("tags").([]any) {
		tags = append(tags, tag.(string))
	}

	err = account.Tags().Set(tags...)
	if err != nil {
		return fmt.Errorf("unable to update tags for account '%s': %s", accountname, err)
	}

	expiry, isSet := d.GetOk("expiry")
	if isSet {
		parsedTime, err := time.Parse(time.RFC3339, expiry.(string))
		if err != nil {
			return fmt.Errorf("unable to parse time string '%s': %s", expiry.(string), err)
		}

		err = account.SetExpiry(parsedTime.Unix())
		if err != nil {
			return fmt.Errorf("unable to set expiry for account '%s': %s", accountname, err)
		}
	}

	err = auth.Commit()
	if err != nil {
		return err
	}

	return nil
}
