package jetstream

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	authb "github.com/synadia-io/jwt-auth-builder.go"
)

func resourceAccountSk() *schema.Resource {
	return &schema.Resource{
		Create: resourceAccountSkCreate,
		Read:   resourceAccountSkRead,
		Delete: resourceAccountSkDelete,
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
			"role": {
				Type:         schema.TypeString,
				Description:  "The role to associate with this key",
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"description": {
				Type:         schema.TypeString,
				Description:  "Description for the signing key",
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"limits": {
				Type:        schema.TypeList,
				Description: "Limits set for the user",
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bearer_tokens": {
							Type:        schema.TypeBool,
							Description: "Description for the signing key",
							Optional:    true,
						},
						"payload": {
							Type:        schema.TypeInt,
							Description: "Maximum allowed payload",
							Optional:    true,
						},
						"subscriptions": {
							Type:        schema.TypeInt,
							Description: "Maximum allowed subscriptions",
							Optional:    true,
						},
						"locale": {
							Type:        schema.TypeString,
							Description: "Locale for the client",
							Optional:    true,
						},
						"connections": {
							Type:        schema.TypeList,
							Description: "Set the allowed connections (nats, ws, wsleaf, mqtt)",
							Optional:    true,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{"nats", "ws", "leaf", "wsleaf", "mqtt"}, false),
							},
						},
					},
				},
			},
			"publish": {
				Type:        schema.TypeList,
				Description: "User's publish permissions",
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"allow": {
							Type:        schema.TypeList,
							Description: "Sets subjects where publishing is allowed",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"deny": {
							Type:        schema.TypeList,
							Description: "Sets subjects where publishing is allowed",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"subscribe": {
				Type:        schema.TypeList,
				Description: "User's subscribe permissions",
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"allow": {
							Type:        schema.TypeList,
							Description: "Sets subjects where publishing is allowed",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"deny": {
							Type:        schema.TypeList,
							Description: "Sets subjects where publishing is allowed",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			// Exported
			"public_key": {
				Type:      schema.TypeString,
				Computed:  true,
				ForceNew:  true,
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
		return fmt.Errorf("operator '%s' not found: %s", operatorname, err)
	}

	accountName := d.Get("account").(string)
	account, err := operator.Accounts().Get(accountName)
	if err != nil {
		return fmt.Errorf("account '%s' not found: %s", accountName, err)
	}

	role, isSet := d.GetOk("role")

	if isSet {
		sk, err := account.ScopedSigningKeys().AddScope(role.(string))
		if err != nil {
			return err
		}

		description, ok := d.GetOk("description")
		if ok {
			sk.SetDescription(description.(string))
		}

		if limits, set := d.GetOk("limits"); set {
			if limitsMap, ok := limits.([]any)[0].(map[string]any); ok {
				err = setKeyLimits(sk, limitsMap)
				if err != nil {
					return fmt.Errorf("unable to set signing key limits: %s", err)
				}
			}
		}

		if publish, set := d.GetOk("publish"); set {
			if publishMap, ok := publish.([]any)[0].(map[string]any); ok {
				err = setPermissions(sk.PubPermissions(), publishMap)
				if err != nil {
					return fmt.Errorf("unable to set publish permissions: %s", err)
				}
			}
		}

		if subscribe, set := d.GetOk("subscribe"); set {
			if subscribeMap, ok := subscribe.([]any)[0].(map[string]any); ok {
				err = setPermissions(sk.SubPermissions(), subscribeMap)
				if err != nil {
					return fmt.Errorf("unable to set publish permissions: %s", err)
				}
			}
		}
		d.Set("public_key", sk.Key())
		d.SetId(sk.Key())
	} else {
		sk, err := account.ScopedSigningKeys().Add()
		if err != nil {
			return fmt.Errorf("unable to create signing key for account %s: %s", accountName, err)
		}
		d.Set("public_key", sk)
		d.SetId(sk)
	}

	err = auth.Commit()
	if err != nil {
		return err
	}

	return nil
}

func resourceAccountSkRead(d *schema.ResourceData, m any) error {
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

	found, _ := account.ScopedSigningKeys().Contains(d.Get("public_key").(string))

	if !found {
		d.SetId("")
	}
	return nil
}

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
	if !found {
		return fmt.Errorf("unable to delete signing key: key not found")
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

func setKeyLimits(scope authb.ScopeLimits, limits map[string]any) error {
	if bearer_tokens, isSet := limits["bearer_tokens"]; isSet {
		err := scope.SetBearerToken(bearer_tokens.(bool))
		if err != nil {
			return err
		}
	}
	if subscriptions, isSet := limits["subscriptions"]; isSet {
		err := scope.SetMaxSubscriptions(int64(subscriptions.(int)))
		if err != nil {
			return err
		}
	}
	if payload, isSet := limits["payload"]; isSet {
		err := scope.SetMaxPayload(int64(payload.(int)))
		if err != nil {
			return err
		}
	}
	if locale, isSet := limits["locale"]; isSet {
		err := scope.SetLocale(locale.(string))
		if err != nil {
			return err
		}
	}
	if connections, isSet := limits["connections"]; isSet {
		c := connections.([]any)
		connectionString := []string{}
		for _, v := range c {
			connectionString = append(connectionString, v.(string))
		}
		err := scope.ConnectionTypes().Set(connectionString...)
		if err != nil {
			return err
		}
	}

	return nil
}

func setPermissions(scopePermissions authb.Permissions, permissions map[string]any) error {
	for _, permType := range []string{"allow", "deny"} {
		if rules, isSet := permissions[permType]; isSet {
			var permissions []string
			for _, rule := range rules.([]any) {
				permissions = append(permissions, rule.(string))
			}
			if permType == "allow" {
				err := scopePermissions.SetAllow(permissions...)
				if err != nil {
					return err
				}
			} else {
				err := scopePermissions.SetDeny(permissions...)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
