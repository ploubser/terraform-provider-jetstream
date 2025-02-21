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

	// check if it's a service export
	for _, exp := range account.Exports().Services().List() {
		if exp.Name() == name {
			d.Set("service", true)
			updateExportResource(exp, d)
			return nil
		}
	}

	// check if it's a stream export
	for _, exp := range account.Exports().Streams().List() {
		if exp.Name() == name {
			updateExportResource(exp, d)
			return nil
		}
	}

	return fmt.Errorf("unable to find export %s in either service exports or stream exports", name)
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

	subject := d.Get("subject").(string)
	name := d.Get("name").(string)
	service, isSet := d.GetOk("service")
	if isSet && service.(bool) {
		found, err := account.Exports().Services().Delete(subject)
		if !found {
			return fmt.Errorf("unable to delete service export %s: export does not exist", name)
		}
		if err != nil {
			return fmt.Errorf("unable to delete service export %s: %s", name, err)
		}

	} else {
		found, err := account.Exports().Streams().Delete(subject)
		if !found {
			return fmt.Errorf("unable to delete stream export %s: export does not exist", name)
		}
		if err != nil {
			return fmt.Errorf("unable to delete stream export %s: %s", name, err)
		}
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
	subject := d.Get("subject").(string)

	export, err := account.Exports().Services().Get(subject)
	if err != nil {
		return fmt.Errorf("failed to load export '%s' for subject '%s': %s", name, subject, err)
	}

	err = updateExportEditableFields(export, d)
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

func updateExportResource(export authb.Export, d *schema.ResourceData) {
	d.Set("subject", export.Subject())
	d.Set("description", export.Description())
	d.Set("url", export.InfoURL())
	d.Set("token_position", export.AccountTokenPosition())
	d.Set("advertise", export.IsAdvertised())
}
