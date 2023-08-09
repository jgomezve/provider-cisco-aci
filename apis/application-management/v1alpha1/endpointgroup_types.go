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

// EndpointGroupParameters are the configurable fields of a EndpointGroup.
type EndpointGroupParameters struct {
	Tenant             string `json:"tenant"`
	ApplicationProfile string `json:"applicationProfile"`
	BridgeDomain       string `json:"bridgeDomain"`
	// +kubebuilder:validation:Optional
	PreferedGroup string `json:"preferedGroup"`
}

// EndpointGroupObservation are the observable fields of a EndpointGroup.
type EndpointGroupObservation struct {
	ObservableField string `json:"observableField,omitempty"`
}

// A EndpointGroupSpec defines the desired state of a EndpointGroup.
type EndpointGroupSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       EndpointGroupParameters `json:"forProvider"`
}

// A EndpointGroupStatus represents the observed state of a EndpointGroup.
type EndpointGroupStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          EndpointGroupObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A EndpointGroup is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,aci}
type EndpointGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EndpointGroupSpec   `json:"spec"`
	Status EndpointGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EndpointGroupList contains a list of EndpointGroup
type EndpointGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EndpointGroup `json:"items"`
}

// EndpointGroup type metadata.
var (
	EndpointGroupKind             = reflect.TypeOf(EndpointGroup{}).Name()
	EndpointGroupGroupKind        = schema.GroupKind{Group: Group, Kind: EndpointGroupKind}.String()
	EndpointGroupKindAPIVersion   = EndpointGroupKind + "." + SchemeGroupVersion.String()
	EndpointGroupGroupVersionKind = SchemeGroupVersion.WithKind(EndpointGroupKind)
)

func init() {
	SchemeBuilder.Register(&EndpointGroup{}, &EndpointGroupList{})
}
