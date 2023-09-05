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

// VrfParameters are the configurable fields of a Vrf.
type VrfParameters struct {
	Name   string `json:"name"`
	Tenant string `json:"tenant"`
	// +kubebuilder:validation:Optional
	NameAlias string `json:"nameAlias"`
}

// VrfObservation are the observable fields of a Vrf.
type VrfObservation struct {
	Dn    string `json:"dn,omitempty"`
	PcTag string `json:"pctag,omitempty"`
}

// A VrfSpec defines the desired state of a Vrf.
type VrfSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       VrfParameters `json:"forProvider"`
}

// A VrfStatus represents the observed state of a Vrf.
type VrfStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          VrfObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Vrf is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="DN",type="string",JSONPath=".status.atProvider.dn",description="Distinguished Name"
// +kubebuilder:printcolumn:name="PCTAG",type="string",JSONPath=".status.atProvider.pctag",description="PcTag"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,aci}
type Vrf struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VrfSpec   `json:"spec"`
	Status VrfStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VrfList contains a list of Vrf
type VrfList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Vrf `json:"items"`
}

// Vrf type metadata.
var (
	VrfKind             = reflect.TypeOf(Vrf{}).Name()
	VrfGroupKind        = schema.GroupKind{Group: Group, Kind: VrfKind}.String()
	VrfKindAPIVersion   = VrfKind + "." + SchemeGroupVersion.String()
	VrfGroupVersionKind = SchemeGroupVersion.WithKind(VrfKind)
)

func init() {
	SchemeBuilder.Register(&Vrf{}, &VrfList{})
}
