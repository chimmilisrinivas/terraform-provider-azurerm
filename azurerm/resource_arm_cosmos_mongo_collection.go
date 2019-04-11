package azurerm

import (
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/cosmos-db/mgmt/2015-04-08/documentdb"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/response"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/validate"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmCosmosMongoCollection() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmCosmosMongoCollectionCreateUpdate,
		Read:   resourceArmCosmosMongoCollectionRead,
		Update: resourceArmCosmosMongoCollectionCreateUpdate,
		Delete: resourceArmCosmosMongoCollectionDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.CosmosEntityName,
			},

			"resource_group_name": resourceGroupNameSchema(),

			"account_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.CosmosAccountName,
			},

			"database_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.CosmosEntityName,
			},

			"indexes": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:         schema.TypeString, // this is a list in the SDK, however any more then a single value causes a 404
							Required:     true,
							ValidateFunc: validate.NoEmptyStrings,
						},
					},

					// expire_after_seconds & unique seem to always cause a 400: Unable to parse request payload
					// so leaving them out.
				},
			},
		},
	}
}

func resourceArmCosmosMongoCollectionCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).cosmosAccountsClient
	ctx := meta.(*ArmClient).StopContext

	name := d.Get("name").(string)
	resourceGroup := d.Get("resource_group_name").(string)
	account := d.Get("account_name").(string)
	database := d.Get("database_name").(string)

	if requireResourcesToBeImported && d.IsNewResource() {
		existing, err := client.GetMongoCollection(ctx, resourceGroup, account, database, name)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("Error checking for presence of creating Cosmos Mongo Collection %s (Account %s, Database %s): %+v", name, account, database, err)
			}
		} else {
			id, err := azure.CosmosGetIDFromResponse(existing.Response)
			if err != nil {
				return fmt.Errorf("Error generating import ID for Cosmos Mongo Collection %s (Account %s, Database %s)", name, account, database)
			}

			return tf.ImportAsExistsError("azurerm_cosmos_mongo_collection", id)
		}
	}

	db := documentdb.MongoCollectionCreateUpdateParameters{
		MongoCollectionCreateUpdateProperties: &documentdb.MongoCollectionCreateUpdateProperties{
			Resource: &documentdb.MongoCollectionResource{
				ID:      &name,
				Indexes: expandCosmosMongoCollectionIndexes(d.Get("indexes")),
			},
			Options: map[string]*string{},
		},
	}

	if d.IsNewResource() {
		future, err := client.CreateMongoCollection(ctx, resourceGroup, account, database, db)
		if err != nil {
			return fmt.Errorf("Error issuing create request for Cosmos Mongo Collection %s (Account %s, Database %s): %+v", name, account, database, err)
		}

		if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
			return fmt.Errorf("Error waiting on create future for Cosmos Mongo Collection %s (Account %s, Database %s): %+v", name, account, database, err)
		}
	} else {
		future, err := client.UpdateMongoCollection(ctx, resourceGroup, account, database, name, db)
		if err != nil {
			return fmt.Errorf("Error issuing update request for Cosmos Mongo Collection %s (Account %s, Database %s): %+v", name, account, database, err)
		}

		if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
			return fmt.Errorf("Error waiting on update future for Cosmos Mongo Collection %s (Account %s, Database %s): %+v", name, account, database, err)
		}
	}

	resp, err := client.GetMongoCollection(ctx, resourceGroup, account, database, name)
	if err != nil {
		return fmt.Errorf("Error making get request for Cosmos Mongo Collection %s (Account %s, Database %s): %+v", name, account, database, err)
	}

	id, err := azure.CosmosGetIDFromResponse(resp.Response)
	if err != nil {
		return fmt.Errorf("Error creating Cosmos Mongo Collection %s (Account %s, Database %s) ID: %v", name, account, database, err)
	}
	d.SetId(id)

	return resourceArmCosmosMongoCollectionRead(d, meta)
}

func resourceArmCosmosMongoCollectionRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).cosmosAccountsClient
	ctx := meta.(*ArmClient).StopContext

	id, err := azure.ParseCosmosDatabaseCollectionID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.GetMongoCollection(ctx, id.ResourceGroup, id.Account, id.Database, id.Collection)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[INFO] Error reading Cosmos Mongo Collection %s (Account %s, Database %s)", id.Collection, id.Account, id.Database)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error reading Cosmos Mongo Collection %s (Account %s, Database %s): %+v", id.Collection, id.Account, id.Database, err)
	}

	if props := resp.MongoCollectionProperties; props != nil {
		d.Set("name", props.ID)
		d.Set("resource_group_name", id.ResourceGroup)
		d.Set("account_name", id.Account)
		d.Set("database_name", id.Database)

		// the API returns this data, but the SDK ignores it?? so lets too
		// looks like they are using key.keys rather then key.key returned by the API
		/*if err := d.Set("indexes", flattenCosmosMongoCollectionIndexes(props.Indexes)); err != nil {
			return fmt.Errorf("Error setting `indexes`: %+v", err)
		}*/
	}

	return nil
}

func resourceArmCosmosMongoCollectionDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).cosmosAccountsClient
	ctx := meta.(*ArmClient).StopContext

	id, err := azure.ParseCosmosDatabaseCollectionID(d.Id())
	if err != nil {
		return err
	}

	future, err := client.DeleteMongoCollection(ctx, id.ResourceGroup, id.Account, id.Database, id.Collection)
	if err != nil {
		if !response.WasNotFound(future.Response()) {
			return fmt.Errorf("Error deleting Cosmos Mongo Collection %s (Account %s, Database %s): %+v", id.Collection, id.Account, id.Database, err)
		}
	}

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err != nil {
		return fmt.Errorf("Error waiting on delete future for Cosmos Mongo Collection %s (Account %s, Database %s): %+v", id.Collection, id.Account, id.Database, err)
	}

	return nil
}

func expandCosmosMongoCollectionIndexes(input interface{}) *[]documentdb.MongoIndex {
	inputs := input.([]interface{})
	outputs := make([]documentdb.MongoIndex, 0)

	for _, i := range inputs {
		b := i.(map[string]interface{})
		outputs = append(outputs, documentdb.MongoIndex{
			Key: &documentdb.MongoIndexKeys{
				Keys: &[]string{b["key"].(string)},
			},
		})
	}

	return &outputs
}

func flattenCosmosMongoCollectionIndexes(indexes *[]documentdb.MongoIndex) *[]map[string]interface{} {
	slice := make([]map[string]interface{}, 0)

	for _, i := range *indexes {
		if key := i.Key; key != nil {
			if keys := key.Keys; keys != nil && len(*keys) > 0 {
				k := (*keys)[0]

				if k != "_id" {
					slice = append(slice, map[string]interface{}{"keys": k})
				}
			}
		}
	}

	return &slice
}
