/*
Copyright 2022 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package endpointgroup

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/connection"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	aciclient "github.com/ciscoecosystem/aci-go-client/v2/client"
	"github.com/ciscoecosystem/aci-go-client/v2/models"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/jgomezve/provider-aci/apis/application-management/v1alpha1"
	apisv1alpha1 "github.com/jgomezve/provider-aci/apis/v1alpha1"
	endpointgrouputil "github.com/jgomezve/provider-aci/internal/clients/endpointgroup"
	"github.com/jgomezve/provider-aci/internal/features"
)

const (
	errNotEndpointGroup = "managed resource is not a EndpointGroup custom resource"
	errTrackPCUsage     = "cannot track ProviderConfig usage"
	errGetPC            = "cannot get ProviderConfig"
	errGetCreds         = "cannot get credentials"

	errNewClient = "cannot create new Service"
)

// A NoOpService does nothing.
type NoOpService struct{}

// var (
// 	newNoOpService = func(_ []byte) (interface{}, error) { return &NoOpService{}, nil }
// )

// Setup adds a controller that reconciles EndpointGroup managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.EndpointGroupGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.EndpointGroupGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: aciclient.NewClient}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.EndpointGroup{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        resource.Tracker
	newServiceFn func(clientUrl, username string, options ...aciclient.Option) *aciclient.Client
}

type SecretData struct {
	Url      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Insecure bool   `json:"insecure"`
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {

	var secretData SecretData
	cr, ok := mg.(*v1alpha1.EndpointGroup)
	if !ok {
		return nil, errors.New(errNotEndpointGroup)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cd := pc.Spec.Credentials
	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	err = json.Unmarshal(data, &secretData)
	if err != nil {
		return nil, err
	}

	svc := c.newServiceFn(secretData.Url, secretData.Username, aciclient.Password(secretData.Password), aciclient.Insecure(secretData.Insecure))
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{apicClient: svc}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// A 'client' used to connect to the external resource API. In practice this
	// would be something like an AWS SDK client.
	apicClient *aciclient.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.EndpointGroup)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotEndpointGroup)
	}

	dn := fmt.Sprintf("uni/tn-%s/ap-%s/epg-%s", cr.Spec.ForProvider.Tenant, cr.Spec.ForProvider.ApplicationProfile, cr.Name)
	fvAEPgCont, err := c.apicClient.Get(dn)

	if err != nil {
		if fmt.Sprintf("%s", err) != "Error retrieving Object: Object may not exist" {
			return managed.ExternalObservation{}, err
		}
	}
	if count := models.G(fvAEPgCont, "totalCount"); count == "0" {
		return managed.ExternalObservation{}, nil
	}
	fvAEPg := models.ApplicationEPGFromContainer(fvAEPgCont)

	if fvAEPg.DistinguishedName == "" {
		return managed.ExternalObservation{}, fmt.Errorf("endpoint group domain %s not found", dn)
	}
	// LateInitializer not required for ACI

	cr.SetConditions(xpv1.Available())
	return managed.ExternalObservation{
		// // Return false when the external resource does not exist. This lets
		// // the managed resource reconciler know that it needs to call Create to
		// // (re)create the resource, or that it has successfully been deleted.
		ResourceExists: true,

		// // Return false when the external resource exists, but it not up to date
		// // with the desired managed resource state. This lets the managed
		// // resource reconciler know that it needs to call Update.
		ResourceUpToDate: endpointgrouputil.IsUptoDate(c.apicClient, cr, fvAEPg.ApplicationEPGAttributes),

		// // Return any details that may be required to connect to the external
		// // resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.EndpointGroup)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotEndpointGroup)
	}

	fmt.Printf("Creating: %+v", cr)
	cr.SetConditions(xpv1.Creating())

	fvAEPgAttr := models.ApplicationEPGAttributes{}
	fvAEPgAttr.PrefGrMemb = cr.Spec.ForProvider.PreferedGroup
	fvAEPg := models.NewApplicationEPG(fmt.Sprintf("epg-%s", cr.Name), fmt.Sprintf("uni/tn-%s/ap-%s", cr.Spec.ForProvider.Tenant, cr.Spec.ForProvider.ApplicationProfile), "", fvAEPgAttr)
	err := c.apicClient.Save(fvAEPg)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "Cannot create Endpoint Group")
	}
	err = c.apicClient.CreateRelationfvRsBdFromApplicationEPG(fvAEPg.DistinguishedName, cr.Spec.ForProvider.BridgeDomain)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "Cannot create association with Bridge Domain")
	}

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.EndpointGroup)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotEndpointGroup)
	}

	fmt.Printf("Updating: %+v", cr)
	fvAEPgAttr := models.ApplicationEPGAttributes{}
	fvAEPgAttr.PrefGrMemb = cr.Spec.ForProvider.PreferedGroup
	fvAEPg := models.NewApplicationEPG(fmt.Sprintf("epg-%s", cr.Name), fmt.Sprintf("uni/tn-%s/ap-%s", cr.Spec.ForProvider.Tenant, cr.Spec.ForProvider.ApplicationProfile), "", fvAEPgAttr)
	fvAEPg.Status = "modified"
	err := c.apicClient.Save(fvAEPg)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "Cannot Update Endpoint Group")
	}
	err = c.apicClient.CreateRelationfvRsBdFromApplicationEPG(fvAEPg.DistinguishedName, cr.Spec.ForProvider.BridgeDomain)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "Cannot update association with Bridge Domain")
	}
	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.EndpointGroup)
	if !ok {
		return errors.New(errNotEndpointGroup)
	}

	cr.SetConditions(xpv1.Deleting())
	dn := fmt.Sprintf("uni/tn-%s/ap-%s/epg-%s", cr.Spec.ForProvider.Tenant, cr.Spec.ForProvider.ApplicationProfile, cr.Name)
	err := c.apicClient.DeleteByDn(dn, "fvAEPg")
	if err != nil {
		return err
	}
	return nil
}
