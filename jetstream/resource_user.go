package jetstream

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	authb "github.com/synadia-io/jwt-auth-builder.go"
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
				Optional:     true,
				ForceNew:     true,
				Computed:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"tags": {
				Type:        schema.TypeList,
				Description: "",
				Optional:    true,
				ForceNew:    false,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"expiry": {
				Type:         schema.TypeString,
				Description:  "Sets an expiration date for the user credentials using a RFC3339 timestamp",
				Optional:     true,
				ForceNew:     false,
				ValidateFunc: validation.IsRFC3339Time,
			},
			"limits": {
				Type:        schema.TypeList,
				Description: "Limits set for the user",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bearer_tokens": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"payload": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"subscriptions": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
			},
			"publish": {
				Type:        schema.TypeList,
				Description: "User's publish permissions",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"allow": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"deny": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"subscribe": {
				Type:        schema.TypeList,
				Description: "User's subscribe permissions",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"allow": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"deny": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},

			// Exported
			"jwt": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
			"credentials": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
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
	operator, err := auth.Operators().Get(opname)
	if err != nil {
		return err
	}

	accountname := d.Get("account").(string)
	account, err := operator.Accounts().Get(accountname)
	if err != nil {
		return fmt.Errorf("cannot find account '%s'. Error: '%s'", accountname, err)
	}

	username := d.Get("name").(string)
	signer, isSet := d.GetOk("signing_key")
	var user authb.User

	if !isSet {
		signer, err = account.ScopedSigningKeys().Add()
		if err != nil {
			return fmt.Errorf("unable to create signing key for user %s: %s", username, err)
		}
	}

	user, err = account.Users().Add(username, signer.(string))
	if err != nil {
		return err
	}

	if !isSet {
		if err = updateUser(user, d); err != nil {
			return fmt.Errorf("cannot update user limits: %s", err)
		}
	}

	// If signer is set, only update limits if key is not scoped
	if _, err = account.ScopedSigningKeys().GetScope(signer.(string)); err != nil {
		if err = updateUser(user, d); err != nil {
			return fmt.Errorf("cannot update user limits: %s", err)
		}
	}

	err = auth.Commit()
	if err != nil {
		return err
	}

	expiry, isSet := d.GetOk("expiry")
	expiryDuration := time.Duration(0)
	if isSet {
		expiresAt, err := time.Parse(time.RFC3339, expiry.(string))
		if err != nil {
			return fmt.Errorf("unable to parse time string '%s': %s", expiry.(string), err)
		}
		expiryDuration = time.Until(expiresAt)
	}

	creds, err := user.Creds(expiryDuration)
	if err != nil {
		return fmt.Errorf("unable to generate creds for user '%s': %s", username, err)
	}

	d.Set("credentials", string(creds))
	d.Set("jwt", user.JWT())
	d.SetId(username)

	return nil
}

func resourceUserRead(d *schema.ResourceData, m any) error {
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

	user, err := account.Users().Get(d.Get("name").(string))
	if err != nil {
		d.SetId("") // Remove from state if user is missing
		return err
	}

	err = d.Set("signing_key", user.Issuer())
	if err != nil {
		return err
	}

	limits := userLimits(user)
	if configuredLimits, set := d.GetOk("limits"); set {
		if limitsMap, ok := configuredLimits.([]any)[0].(map[string]any); ok {
			for k := range limitsMap {
				limitsMap[k] = limits[k]
			}
			err = d.Set("limits", []any{limitsMap})
			if err != nil {
				return err
			}
		}
	}

	pubPerms := publishPermissions(user)
	if configuredPubPerm, set := d.GetOk("publish"); set {
		if pubPermsMap, ok := configuredPubPerm.([]any)[0].(map[string]any); ok {
			for k := range pubPermsMap {
				pubPermsMap[k] = pubPerms[k]
			}
			err = d.Set("publish", []any{pubPermsMap})
			if err != nil {
				return err
			}
		}
	}

	subPerms := subscribePermissions(user)
	if configuredSubPerm, set := d.GetOk("publish"); set {
		if subPermsMap, ok := configuredSubPerm.([]any)[0].(map[string]any); ok {
			for k := range subPermsMap {
				subPermsMap[k] = subPerms[k]
			}
			err = d.Set("subscribe", []any{subPermsMap})
			if err != nil {
				return err
			}
		}
	}

	tags, err := user.Tags().All()
	if err != nil {
		return err
	}

	if err := d.Set("tags", tags); err != nil {
		return err
	}

	if err := d.Set("jwt", user.JWT()); err != nil {
		return err
	}

	return nil
}

func userLimits(user authb.User) map[string]any {
	return map[string]any{
		"bearer_tokens": user.BearerToken(),
		"payload":       user.MaxPayload(),
		"subscriptions": user.MaxSubscriptions(),
	}
}

func publishPermissions(user authb.User) map[string]any {
	return map[string]any{
		"allow": user.PubPermissions().Allow(),
		"deny":  user.PubPermissions().Deny(),
	}
}

func subscribePermissions(user authb.User) map[string]any {
	return map[string]any{
		"allow": user.SubPermissions().Allow(),
		"deny":  user.SubPermissions().Deny(),
	}
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

	err = auth.Commit()
	if err != nil {
		return err
	}

	return nil
}

func updateUser(user authb.User, d *schema.ResourceData) error {
	if limits, set := d.GetOk("limits"); set {
		if limitsMap, ok := limits.([]any)[0].(map[string]any); ok {
			if bearer_tokens, isSet := limitsMap["bearer_tokens"]; isSet {
				err := user.SetBearerToken(bearer_tokens.(bool))
				if err != nil {
					return err
				}
			}
			if subscriptions, isSet := limitsMap["subscriptions"]; isSet {
				err := user.SetMaxSubscriptions(int64(subscriptions.(int)))
				if err != nil {
					return err
				}
			}
			if payload, isSet := limitsMap["payload"]; isSet {
				err := user.SetMaxPayload(int64(payload.(int)))
				if err != nil {
					return err
				}
			}
		}
	}

	if publish, set := d.GetOk("publish"); set {
		if publishMap, ok := publish.([]any)[0].(map[string]any); ok {
			for _, permType := range []string{"allow", "deny"} {
				if rules, isSet := publishMap[permType]; isSet {
					var permissions []string
					for _, rule := range rules.([]any) {
						permissions = append(permissions, rule.(string))
					}
					if permType == "allow" {
						err := user.PubPermissions().SetAllow(permissions...)
						if err != nil {
							return err
						}
					} else {
						err := user.PubPermissions().SetDeny(permissions...)
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}

	if subscribe, set := d.GetOk("subscribe"); set {
		if subscribeMap, ok := subscribe.([]any)[0].(map[string]any); ok {
			for _, permType := range []string{"allow", "deny"} {
				if rules, isSet := subscribeMap[permType]; isSet {
					var permissions []string
					for _, rule := range rules.([]any) {
						permissions = append(permissions, rule.(string))
					}
					if permType == "allow" {
						err := user.SubPermissions().SetAllow(permissions...)
						if err != nil {
							return err
						}
					} else {
						err := user.SubPermissions().SetDeny(permissions...)
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}

	return nil
}

func resourceUserUpdate(d *schema.ResourceData, m any) error {
	conf := m.(ProviderConfig)
	auth, err := NewAuthProvider(conf)
	if err != nil {
		return err
	}

	opname := d.Get("operator").(string)
	operator, err := auth.Operators().Get(opname)
	if err != nil {
		return err
	}

	accountname := d.Get("account").(string)
	account, err := operator.Accounts().Get(accountname)
	if err != nil {
		return fmt.Errorf("cannot find account '%s'. Error: '%s'", accountname, err)
	}

	user, err := account.Users().Get(d.Get("name").(string))
	if err != nil {
		return err
	}

	err = updateUser(user, d)
	if err != nil {
		return err
	}

	err = auth.Commit()
	if err != nil {
		return err
	}

	return nil
}
