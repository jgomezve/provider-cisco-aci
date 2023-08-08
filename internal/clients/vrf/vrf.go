package vrf

import (
	"github.com/ciscoecosystem/aci-go-client/v2/models"
	"github.com/google/go-cmp/cmp"
	"github.com/jgomezve/provider-aci/apis/networking/v1alpha1"
)

func IsUptoDate(s v1alpha1.VrfParameters, t models.VRFAttributes) bool {
	observed := &v1alpha1.VrfParameters{
		Name:      t.Name,
		NameAlias: t.NameAlias,
	}

	return cmp.Equal(observed, &s)
}
