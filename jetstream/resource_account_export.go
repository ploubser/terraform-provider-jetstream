package jetstream

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	authb "github.com/synadia-io/jwt-auth-builder.go"
)

func resourceAccountExport() *schema.Resource {
	return &schema.Resource{
		Create: resourceAccountExportCreate,
		Read:   resourceAccountExportRead,
		Delete: resourceAccountExportDelete,
		Update: resourceAccountExportUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Description:  "The name of the export",
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
			"subject": {
				Type:        schema.TypeString,
				Description: "The Subject to export",
				Required:    true,
				ForceNew:    true,
			},
			"service": {
				Type:        schema.TypeBool,
				Description: "Sets the Export to be a Service rather than a Stream",
				Optional:    true,
				ForceNew:    true,
			},
			"description": {
				Type:        schema.TypeString,
				Description: "Friendly description",
				Optional:    true,
				ForceNew:    false,
			},
			"url": {
				Type:        schema.TypeString,
				Description: "Sets a URL for further information",
				Optional:    true,
				ForceNew:    false,
			},
			"token_position": {
				Type:         schema.TypeInt,
				Description:  "The position to use for the Account name",
				Optional:     true,
				ForceNew:     false,
				ValidateFunc: validation.IntAtLeast(0),
			},
			"advertise": {
				Type:        schema.TypeBool,
				Description: "Advertise the Export",
				Optional:    true,
				ForceNew:    false,
			},
		},
	}
}

func resourceAccountExportCreate(d *schema.ResourceData, m any) error {
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

	accountname := d.Get("account").(string)
	account, err := operator.Accounts().Get(accountname)
	if err != nil {
		return fmt.Errorf("failed to load account '%s: %s", accountname, err)
	}

	name := d.Get("name").(string)
	subject := d.Get("subject").(string)

	var export authb.Export
	service, isSet := d.GetOk("service")
	if isSet && service.(bool) {
		export, err = account.Exports().Services().Add(name, subject)
		if err != nil {
			return fmt.Errorf("unable to create service export %s: %s", name, err)
		}

	} else {
		export, err = account.Exports().Streams().Add(name, subject)
		if err != nil {
			return fmt.Errorf("unable to create stream export %s: %s", name, err)
		}
	}

	err = updateExportEditableFields(export, d)
	if err != nil {
		return err
	}

	err = auth.Commit()
	if err != nil {
		return fmt.Errorf("unable to commit changes to export %s: %s", name, err)
	}

	d.SetId(name)

	return nil
}

func resourceAccountExportRead(d *schema.ResourceData, m any) error {
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

	accountname := d.Get("account").(string)
	account, err := operator.Accounts().Get(accountname)
	if err != nil {
		return fmt.Errorf("failed to load account '%s: %s", accountname, err)
	}

	name := d.Get("name").(string)
	var exp authb.Export

	exp, err = account.Exports().Services().GetByName(name)
	if err != nil && err != authb.ErrNotFound {
		return fmt.Errorf("unable to load service export %s: %s", name, err)
	}

	if err == authb.ErrNotFound {
		exp, err = account.Exports().Streams().GetByName(name)
		if err != nil {
			if err == authb.ErrNotFound {
				d.SetId("")
				return fmt.Errorf("unable to find service or stream export %s", name)
			}
			return fmt.Errorf("unable to load stream export %s: %s", name, err)
		}
	} else {
		d.Set("service", true)
	}

	err = updateExportResource(exp, d)
	if err != nil {
		return err
	}

	return nil
}

func resourceAccountExportDelete(d *schema.ResourceData, m any) error {
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

	accountname := d.Get("account").(string)
	account, err := operator.Accounts().Get(accountname)
	if err != nil {
		return fmt.Errorf("failed to load account '%s: %s", accountname, err)
	}

	name := d.Get("name").(string)
	subject := d.Get("subject").(string)

	var found bool
	if d.Get("service").(bool) {
		found, err = account.Exports().Services().Delete(subject)
	} else {
		found, err = account.Exports().Streams().Delete(subject)
	}

	if !found {
		return fmt.Errorf("could not find export %s with subject %s", name, subject)
	}
	if err != nil {
		return fmt.Errorf("unable to delete export %s: %s", name, err)
	}

	err = auth.Commit()
	if err != nil {
		return fmt.Errorf("unable to commit changes to export %s: %s", name, err)
	}

	return nil
}

func resourceAccountExportUpdate(d *schema.ResourceData, m any) error {
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

	accountname := d.Get("account").(string)
	account, err := operator.Accounts().Get(accountname)
	if err != nil {
		return fmt.Errorf("failed to load account '%s: %s", accountname, err)
	}

	name := d.Get("name").(string)
	var exp authb.Export

	exp, err = account.Exports().Services().GetByName(name)
	if err != nil && err != authb.ErrNotFound {
		return fmt.Errorf("unable to load service export %s: %s", name, err)
	}

	if err == authb.ErrNotFound {
		exp, err = account.Exports().Streams().GetByName(name)
		if err != nil {
			if err == authb.ErrNotFound {
				return fmt.Errorf("unable to find service or stream export %s", name)
			}
			return fmt.Errorf("unable to load stream export %s: %s", name, err)
		}
	}

	err = updateExportEditableFields(exp, d)
	if err != nil {
		return err
	}

	err = auth.Commit()
	if err != nil {
		return fmt.Errorf("unable to commit changes to export %s: %s", name, err)
	}

	return nil
}

func updateExportEditableFields(export authb.Export, d *schema.ResourceData) error {
	fields := map[string]func(any) error{
		"description":    func(v any) error { return export.SetDescription(v.(string)) },
		"url":            func(v any) error { return export.SetInfoURL(v.(string)) },
		"token_position": func(v any) error { return export.SetAccountTokenPosition(uint(v.(int))) },
		"advertise":      func(v any) error { return export.SetAdvertised(v.(bool)) },
	}

	for key, setter := range fields {
		if value, isSet := d.GetOk(key); isSet {
			if err := setter(value); err != nil {
				return fmt.Errorf("unable to set %s for export %s: %s", key, export.Name(), err)
			}
		}
	}
	return nil
}

func updateExportResource(export authb.Export, d *schema.ResourceData) error {
	fields := map[string]any{
		"subject":        export.Subject(),
		"description":    export.Description(),
		"url":            export.InfoURL(),
		"token_position": export.AccountTokenPosition(),
		"advertise":      export.IsAdvertised(),
	}

	for key, value := range fields {
		if err := d.Set(key, value); err != nil {
			return fmt.Errorf("unable to set %s field: %w", key, err)
		}
	}

	return nil
}
