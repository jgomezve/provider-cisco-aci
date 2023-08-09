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

// BridgeDomainParameters are the configurable fields of a BridgeDomain.
type BridgeDomainParameters struct {
	Tenant   string `json:"tenant"`
	Vrf      string `json:"vrf"`
	ArpFlood string `json:"arpFlood"`
}

// BridgeDomainObservation are the observable fields of a BridgeDomain.
type BridgeDomainObservation struct {
	ObservableField string `json:"observableField,omitempty"`
}

// A BridgeDomainSpec defines the desired state of a BridgeDomain.
type BridgeDomainSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       BridgeDomainParameters `json:"forProvider"`
}

// A BridgeDomainStatus represents the observed state of a BridgeDomain.
type BridgeDomainStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          BridgeDomainObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A BridgeDomain is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,aci}
type BridgeDomain struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BridgeDomainSpec   `json:"spec"`
	Status BridgeDomainStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BridgeDomainList contains a list of BridgeDomain
type BridgeDomainList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BridgeDomain `json:"items"`
}

// BridgeDomain type metadata.
var (
	BridgeDomainKind             = reflect.TypeOf(BridgeDomain{}).Name()
	BridgeDomainGroupKind        = schema.GroupKind{Group: Group, Kind: BridgeDomainKind}.String()
	BridgeDomainKindAPIVersion   = BridgeDomainKind + "." + SchemeGroupVersion.String()
	BridgeDomainGroupVersionKind = SchemeGroupVersion.WithKind(BridgeDomainKind)
)

func init() {
	SchemeBuilder.Register(&BridgeDomain{}, &BridgeDomainList{})
}
