package jetstream

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	authb "github.com/synadia-io/jwt-auth-builder.go"
)

func resourceImport() *schema.Resource {
	return &schema.Resource{
		Create: resourceImportCreate,
		Read:   resourceImportRead,
		Delete: resourceImportDelete,
		Update: resourceImportUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Description:  "The import name",
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
				Description: "The subject we are going to import",
				Required:    true,
				ForceNew:    true,
			},
			"source": {
				Type:        schema.TypeString,
				Description: "The account public key to import from",
				Required:    true,
				ForceNew:    true,
			},
			"service": {
				Type:        schema.TypeBool,
				Description: "Sets the import to be a Service rather than a Stream",
				Optional:    true,
				ForceNew:    true,
			},
			"local": {
				Type:        schema.TypeString,
				Description: "The local Subject to use for the import",
				Optional:    true,
				ForceNew:    false,
			},
			"traceable": {
				Type:        schema.TypeBool,
				Description: "Enable tracing messages across Stream imports",
				Optional:    true,
				ForceNew:    false,
			},
			"share": {
				Type:        schema.TypeBool,
				Description: "Shares connection information with the exporter",
				Optional:    true,
				ForceNew:    false,
			},
		},
	}
}

func resourceImportCreate(d *schema.ResourceData, m any) error {
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

	var imp authb.Import
	name, subject := d.Get("name").(string), d.Get("subject").(string)
	isService := d.Get("service").(bool)

	if isService {
		imp, err = account.Imports().Services().Add(name, account.Subject(), subject)
	} else {
		imp, err = account.Imports().Streams().Add(name, account.Subject(), subject)
	}
	if err != nil {
		return fmt.Errorf("unable to create import %s: %s", name, err)
	}

	sourceAccount, err := operator.Accounts().Get(d.Get("source").(string))
	if err != nil {
		return fmt.Errorf("failed to load source account '%s: %s", sourceAccount, err)
	}

	err = imp.SetAccount(sourceAccount.Subject())
	if err != nil {
		return fmt.Errorf("unable to set account key for import %s: %s", name, err)
	}

	err = updateImportEditableFields(imp, d, isService, subject)
	if err != nil {
		return err
	}

	err = auth.Commit()
	if err != nil {
		return fmt.Errorf("unable to commit changes to import %s: %s", name, err)
	}

	d.SetId(name)

	return nil
}

func resourceImportRead(d *schema.ResourceData, m any) error {
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

	var imp authb.Import
	isService := d.Get("service").(bool)
	name := d.Get("name").(string)
	if isService {
		imp, err = account.Imports().Services().GetByName(name)
	} else {
		imp, err = account.Imports().Streams().GetByName(name)
	}
	if err == authb.ErrNotFound {
		d.SetId("")
		return nil
	} else if err != nil {
		return fmt.Errorf("unable to read import %s: %s", name, err)
	}

	err = updateImportResource(imp, d)
	if err != nil {
		return err
	}

	return nil
}

func resourceImportDelete(d *schema.ResourceData, m any) error {
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

	var imp authb.Import
	isService := d.Get("service").(bool)
	name := d.Get("name").(string)
	if isService {
		imp, err = account.Imports().Services().GetByName(name)
	} else {
		imp, err = account.Imports().Streams().GetByName(name)
	}
	if err != nil {
		return fmt.Errorf("unable to read import %s: %s", name, err)
	}

	var found bool
	if isService {
		found, err = account.Imports().Services().Delete(imp.Subject())
	} else {
		found, err = account.Imports().Streams().Delete(imp.Subject())
	}

	if !found {
		return fmt.Errorf("could not find import %s", name)
	}
	if err != nil {
		return fmt.Errorf("unable to delete import %s: %s", name, err)
	}

	err = auth.Commit()
	if err != nil {
		return fmt.Errorf("unable to commit changes to import %s: %s", name, err)
	}

	return nil
}

func resourceImportUpdate(d *schema.ResourceData, m any) error {
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

	var imp authb.Import
	name, subject := d.Get("name").(string), d.Get("subject").(string)
	isService := d.Get("service").(bool)

	if isService {
		imp, err = account.Imports().Services().GetByName(name)
	} else {
		imp, err = account.Imports().Streams().GetByName(name)
	}
	if err != nil {
		return fmt.Errorf("unable to read import %s: %s", name, err)
	}

	err = updateImportEditableFields(imp, d, isService, subject)
	if err != nil {
		return err
	}

	err = auth.Commit()
	if err != nil {
		return fmt.Errorf("unable to commit changes to import %s: %s", name, err)
	}

	return nil
}

func updateImportEditableFields(imp authb.Import, d *schema.ResourceData, isService bool, subject string) error {
	if localSub, ok := d.GetOk("local"); ok {
		if err := imp.SetLocalSubject(localSub.(string)); err != nil {
			return fmt.Errorf("unable to set local subject: %w", err)
		}
	} else {
		imp.SetLocalSubject(subject)
	}

	if share, _ := d.GetOk("share"); share.(bool) {
		if err := imp.SetShareConnectionInfo(true); err != nil {
			return fmt.Errorf("unable to set ShareConnectionInfo: %w", err)
		}
	}

	if allowTrace, _ := d.GetOk("traceable"); allowTrace.(bool) && !isService {
		if err := imp.SetShareConnectionInfo(true); err != nil {
			return fmt.Errorf("unable to set allow trace: %w", err)
		}
	}

	return nil
}

func updateImportResource(imp authb.Import, d *schema.ResourceData) error {
	fields := map[string]any{
		"subject":   imp.Subject(),
		"local":     imp.LocalSubject(),
		"share":     imp.IsShareConnectionInfo(),
		"traceable": imp.IsShareConnectionInfo(),
	}

	for key, value := range fields {
		if err := d.Set(key, value); err != nil {
			return fmt.Errorf("unable to set %s field: %w", key, err)
		}
	}

	return nil
}
