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

package bridgedomain

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
	"github.com/jgomezve/provider-aci/apis/networking/v1alpha1"
	apisv1alpha1 "github.com/jgomezve/provider-aci/apis/v1alpha1"
	bridgedomainutil "github.com/jgomezve/provider-aci/internal/clients/bridgedomain"
	"github.com/jgomezve/provider-aci/internal/features"
)

const (
	errNotBridgeDomain = "managed resource is not a BridgeDomain custom resource"
	errTrackPCUsage    = "cannot track ProviderConfig usage"
	errGetPC           = "cannot get ProviderConfig"
	errGetCreds        = "cannot get credentials"

	errNewClient = "cannot create new Service"
)

// A NoOpService does nothing.
type NoOpService struct{}

// var (
// 	newNoOpService = func(_ []byte) (interface{}, error) { return &NoOpService{}, nil }
// )

// Setup adds a controller that reconciles BridgeDomain managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.BridgeDomainGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.BridgeDomainGroupVersionKind),
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
		For(&v1alpha1.BridgeDomain{}).
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
	cr, ok := mg.(*v1alpha1.BridgeDomain)
	if !ok {
		return nil, errors.New(errNotBridgeDomain)
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
	cr, ok := mg.(*v1alpha1.BridgeDomain)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotBridgeDomain)
	}

	dn := fmt.Sprintf("uni/tn-%s/BD-%s", cr.Spec.ForProvider.Tenant, cr.Spec.ForProvider.Name)
	fvBdCont, err := c.apicClient.Get(dn)

	if err != nil {
		if fmt.Sprintf("%s", err) != "Error retrieving Object: Object may not exist" {
			return managed.ExternalObservation{}, err
		}
	}
	if count := models.G(fvBdCont, "totalCount"); count == "0" {
		return managed.ExternalObservation{}, nil
	}
	fvBd := models.BridgeDomainFromContainer(fvBdCont)

	if fvBd.DistinguishedName == "" {
		return managed.ExternalObservation{}, fmt.Errorf("bridge domain %s not found", dn)
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
		ResourceUpToDate: bridgedomainutil.IsUptoDate(c.apicClient, cr, fvBd.BridgeDomainAttributes),

		// // Return any details that may be required to connect to the external
		// // resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.BridgeDomain)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotBridgeDomain)
	}

	fmt.Printf("Creating: %+v", cr)
	cr.SetConditions(xpv1.Creating())

	fvBdAttr := models.BridgeDomainAttributes{}
	fvBdAttr.ArpFlood = cr.Spec.ForProvider.ArpFlood
	fvBd := models.NewBridgeDomain(fmt.Sprintf("BD-%s", cr.Spec.ForProvider.Name), fmt.Sprintf("uni/tn-%s", cr.Spec.ForProvider.Tenant), "", fvBdAttr)
	err := c.apicClient.Save(fvBd)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "Cannot create Bridge Domain")
	}
	err = c.apicClient.CreateRelationfvRsCtxFromBridgeDomain(fvBd.DistinguishedName, cr.Spec.ForProvider.Vrf)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "Cannot create association with VRF")
	}

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.BridgeDomain)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotBridgeDomain)
	}

	fmt.Printf("Updating: %+v", cr)
	fvBdAttr := models.BridgeDomainAttributes{}
	fvBdAttr.ArpFlood = cr.Spec.ForProvider.ArpFlood
	fvBd := models.NewBridgeDomain(fmt.Sprintf("BD-%s", cr.Spec.ForProvider.Name), fmt.Sprintf("uni/tn-%s", cr.Spec.ForProvider.Tenant), "", fvBdAttr)
	fvBd.Status = "modified"
	err := c.apicClient.Save(fvBd)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "Cannot update Bridge Domain")
	}
	err = c.apicClient.CreateRelationfvRsCtxFromBridgeDomain(fvBd.DistinguishedName, cr.Spec.ForProvider.Vrf)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "Cannot update association with VRF")
	}
	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.BridgeDomain)
	if !ok {
		return errors.New(errNotBridgeDomain)
	}

	cr.SetConditions(xpv1.Deleting())
	dn := fmt.Sprintf("uni/tn-%s/BD-%s", cr.Spec.ForProvider.Tenant, cr.Spec.ForProvider.Name)
	err := c.apicClient.DeleteByDn(dn, "fvBD")
	if err != nil {
		return err
	}
	return nil
}
