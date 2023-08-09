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

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// ApplicationProfileParameters are the configurable fields of a ApplicationProfile.
type ApplicationProfileParameters struct {
	Tenant string `json:"tenant"`
	// +kubebuilder:validation:Optional
	NameAlias string `json:"nameAlias"`
}

// ApplicationProfileObservation are the observable fields of a ApplicationProfile.
type ApplicationProfileObservation struct {
	ObservableField string `json:"observableField,omitempty"`
}

// A ApplicationProfileSpec defines the desired state of a ApplicationProfile.
type ApplicationProfileSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ApplicationProfileParameters `json:"forProvider"`
}

// A ApplicationProfileStatus represents the observed state of a ApplicationProfile.
type ApplicationProfileStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ApplicationProfileObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A ApplicationProfile is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,aci}
type ApplicationProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationProfileSpec   `json:"spec"`
	Status ApplicationProfileStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationProfileList contains a list of ApplicationProfile
type ApplicationProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApplicationProfile `json:"items"`
}

// ApplicationProfile type metadata.
var (
	ApplicationProfileKind             = reflect.TypeOf(ApplicationProfile{}).Name()
	ApplicationProfileGroupKind        = schema.GroupKind{Group: Group, Kind: ApplicationProfileKind}.String()
	ApplicationProfileKindAPIVersion   = ApplicationProfileKind + "." + SchemeGroupVersion.String()
	ApplicationProfileGroupVersionKind = SchemeGroupVersion.WithKind(ApplicationProfileKind)
)

func init() {
	SchemeBuilder.Register(&ApplicationProfile{}, &ApplicationProfileList{})
}
