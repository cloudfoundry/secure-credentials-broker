package broker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-incubator/credhub-cli/credhub"
	"github.com/cloudfoundry-incubator/credhub-cli/credhub/credentials/values"
	"github.com/cloudfoundry-incubator/credhub-cli/credhub/permissions"
	"github.com/pivotal-cf/brokerapi"
)

const (
	PlanNameDefault = "default"
	BrokerID        = "secure-credentials-broker"
	ServiceID       = "secure-credentials"
	CredentialsID   = "credentials"
)

type InstanceCredentials struct {
	Host     string
	Port     int
	Password string
}

type InstanceCreator interface {
	Create() error
	Destroy() error
	InstanceExists() (bool, error)
}

type InstanceBinder interface {
	Bind() (InstanceCredentials, error)
	Unbind() error
	InstanceExists() (bool, error)
}

type CredhubServiceBroker struct {
	InstanceCreators map[string]InstanceCreator
	InstanceBinders  map[string]InstanceBinder
	CredHubClient    *credhub.CredHub
	Logger           lager.Logger
}

func (credhubServiceBroker *CredhubServiceBroker) Services(context context.Context) []brokerapi.Service {
	planList := []brokerapi.ServicePlan{}
	for _, plan := range credhubServiceBroker.plans() {
		planList = append(planList, *plan)
	}

	return []brokerapi.Service{
		brokerapi.Service{
			ID:            ServiceID,
			Name:          ServiceID,
			Description:   "Stores configuration parameters securely in CredHub",
			Bindable:      true,
			PlanUpdatable: true,
			Plans:         planList,
			Metadata: &brokerapi.ServiceMetadata{
				DisplayName:         "credhub-broker",
				LongDescription:     "Stores configuration parameters securely in CredHub",
				DocumentationUrl:    "",
				SupportUrl:          "",
				ImageUrl:            "",
				ProviderDisplayName: "",
			},
			Tags: []string{
				"credhub",
			},
		},
	}
}

func (credhubServiceBroker *CredhubServiceBroker) Provision(context context.Context, instanceID string, serviceDetails brokerapi.ProvisionDetails, asyncAllowed bool) (spec brokerapi.ProvisionedServiceSpec, err error) {
	err = credhubServiceBroker.setJSON(serviceDetails.RawParameters, instanceID, serviceDetails.ServiceID)

	if err != nil {
		return spec, err
	}

	credhubServiceBroker.Logger.Info("successfully stored user-provided credentials for instanceID " + instanceID)
	return spec, nil
}

func (credhubServiceBroker *CredhubServiceBroker) Deprovision(context context.Context, instanceID string, details brokerapi.DeprovisionDetails, asyncAllowed bool) (brokerapi.DeprovisionServiceSpec, error) {
	serviceInstanceKey := constructKey(details.ServiceID, instanceID, CredentialsID)

	err := credhubServiceBroker.CredHubClient.Delete(serviceInstanceKey)
	if err != nil {
		return brokerapi.DeprovisionServiceSpec{}, err
	}
	// TODO do we need to delete or check for orphaned actor entries?

	credhubServiceBroker.Logger.Info("successfully deprovisioned service instance key" + serviceInstanceKey)
	return brokerapi.DeprovisionServiceSpec{}, nil
}

func (credhubServiceBroker *CredhubServiceBroker) Bind(context context.Context, instanceID, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, error) {
	if details.BindResource.AppGuid == "" {
		return brokerapi.Binding{}, errors.New("No app-guid was provided in the binding request, you must have one")
	}

	// TODO do we need some kind of transcation for this?
	actor := fmt.Sprintf("mtls-app:%s", details.BindResource.AppGuid)
	bindingKey := constructKey(details.ServiceID, instanceID, bindingID)
	_, err := credhubServiceBroker.CredHubClient.SetValue(bindingKey, values.Value(actor), credhub.Mode("no-overwrite"))
	if err != nil {
		return brokerapi.Binding{}, err
	}

	additionalPermissions := []permissions.Permission{
		{
			Actor:      actor,
			Operations: []string{"read"},
		},
	}
	key := constructKey(details.ServiceID, instanceID, CredentialsID)
	_, err = credhubServiceBroker.CredHubClient.AddPermissions(key, additionalPermissions)
	if err != nil {
		return brokerapi.Binding{}, err
	}

	credhubServiceBroker.Logger.Info("successfully bound service instance for key " + bindingKey)
	return brokerapi.Binding{Credentials: map[string]string{"credhub-ref": key}}, nil
}

func (credhubServiceBroker *CredhubServiceBroker) Unbind(context context.Context, instanceID, bindingID string, details brokerapi.UnbindDetails) error {
	bindingKey := constructKey(details.ServiceID, instanceID, bindingID)

	// TODO transaction?
	credhubServiceBroker.Logger.Info("retrieving service binding actor for key " + bindingKey)
	actor, err := credhubServiceBroker.CredHubClient.GetLatestValue(bindingKey)
	if err != nil {
		return err
	}

	serviceInstanceKey := constructKey(details.ServiceID, instanceID, CredentialsID)
	credhubServiceBroker.Logger.Info("deleting permissions for actor and key", lager.Data{"actor": string(actor.Value), "key": serviceInstanceKey})
	credhubServiceBroker.deletePermissions(serviceInstanceKey, string(actor.Value))
	if err != nil {
		return err
	}

	credhubServiceBroker.Logger.Info("deleting binding for key", lager.Data{"key": bindingKey})
	credhubServiceBroker.CredHubClient.Delete(bindingKey)
	if err != nil {
		return err
	}

	return nil
}

// LastOperation ...
func (credhubServiceBroker *CredhubServiceBroker) LastOperation(context context.Context, instanceID, operationData string) (brokerapi.LastOperation, error) {
	return brokerapi.LastOperation{}, nil
}

func (credhubServiceBroker *CredhubServiceBroker) Update(context context.Context, instanceID string, serviceDetails brokerapi.UpdateDetails, asyncAllowed bool) (spec brokerapi.UpdateServiceSpec, err error) {
	err = credhubServiceBroker.setJSON(serviceDetails.RawParameters, instanceID, serviceDetails.ServiceID)

	if err != nil {
		return spec, err
	}

	credhubServiceBroker.Logger.Info("successfully updated user-provided credentials for instance " + instanceID)
	return spec, nil
}

func (credhubServiceBroker *CredhubServiceBroker) plans() map[string]*brokerapi.ServicePlan {
	plans := map[string]*brokerapi.ServicePlan{}

	plans[PlanNameDefault] = &brokerapi.ServicePlan{
		ID:          PlanNameDefault,
		Name:        PlanNameDefault,
		Description: "Stores configuration parameters securely in CredHub",
		Metadata: &brokerapi.ServicePlanMetadata{
			Bullets: []string{
				"Stores configuration parameters securely in CredHub",
			},
			DisplayName: PlanNameDefault,
		},
	}

	return plans
}

func constructKey(serviceID, instanceID, suffixID string) string {
	return fmt.Sprintf("/c/%s/%s/%s/%s", BrokerID, serviceID, instanceID, suffixID)
}

func (credhubServiceBroker *CredhubServiceBroker) setJSON(rawParameters json.RawMessage, instanceID string, serviceID string) (err error) {
	var credentials map[string]interface{}
	err = json.Unmarshal(rawParameters, &credentials)
	if err != nil {
		return brokerapi.ErrRawParamsInvalid
	}

	key := constructKey(serviceID, instanceID, CredentialsID)
	_, err = credhubServiceBroker.CredHubClient.SetJSON(key, values.JSON(credentials), credhub.Mode("overwrite"))

	if err != nil {
		credhubServiceBroker.Logger.Error("unable to store user-provided credentials to credhub ", err, map[string]interface{}{"key": key})
		return brokerapi.NewFailureResponse(err, http.StatusInternalServerError, "unable to store the user-provided credentials")
	}

	return nil
}

func (credhubServiceBroker *CredhubServiceBroker) deletePermissions(credName string, actor string) error {
	query := url.Values{}
	query.Set("credential_name", credName)
	query.Set("actor", actor)

	_, err := credhubServiceBroker.CredHubClient.Request(http.MethodDelete, "/api/v1/permissions", query, nil)
	if err != nil {
		return err
	}

	return nil
}
