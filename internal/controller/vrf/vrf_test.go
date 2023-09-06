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

package vrf

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	aciclient "github.com/ciscoecosystem/aci-go-client/v2/client"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/jgomezve/provider-aci/apis/networking/v1alpha1"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

func TestObserve(t *testing.T) {
	type fields struct {
		// service interface{}
	}

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o   managed.ExternalObservation
		err error
	}

	cases := map[string]struct {
		handler http.Handler
		reason  string
		fields  fields
		args    args
		want    want
	}{
		"VrfNotFound": {
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = r.Body.Close()
				if diff := cmp.Diff(http.MethodGet, r.Method); diff != "" {
					t.Errorf("r: -want, +got:\n%s", diff)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"totalCount":"0","imdata":[]}`))

			}),
			args: args{
				mg: &v1alpha1.Vrf{Spec: v1alpha1.VrfSpec{ForProvider: v1alpha1.VrfParameters{Name: "test"}}},
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: nil,
			},
			reason: "VRF does not exist",
		},
		"VrfUpdated": {
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = r.Body.Close()
				if diff := cmp.Diff(http.MethodGet, r.Method); diff != "" {
					t.Errorf("r: -want, +got:\n%s", diff)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"totalCount":"1","imdata":[{"fvCtx":{"attributes":{"name":"Test","nameAlias":"TestAlias"}}}]}`))

			}),
			args: args{
				mg: &v1alpha1.Vrf{
					Spec: v1alpha1.VrfSpec{
						ForProvider: v1alpha1.VrfParameters{
							Name:      "Test",
							Tenant:    "Test",
							NameAlias: "TestAlias",
						},
					},
				},
			},
			want: want{
				o:   managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ConnectionDetails: managed.ConnectionDetails{}},
				err: nil,
			},
			reason: "VRF does is updated",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()
			ac := aciclient.NewClient(server.URL, "test_user", aciclient.Password("test_password"), aciclient.SkipLoggingPayload(true))
			ac.AuthToken = &aciclient.Auth{Token: "test", Expiry: time.Now().Add(1 * time.Hour)}
			e := external{apicClient: ac}
			got, err := e.Observe(tc.args.ctx, tc.args.mg)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\ne.Observe(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\ne.Observe(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}
