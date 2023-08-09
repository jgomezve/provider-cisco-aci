package applicationprofile

import (
	"github.com/ciscoecosystem/aci-go-client/v2/models"
	"github.com/google/go-cmp/cmp"
	"github.com/jgomezve/provider-aci/apis/application-management/v1alpha1"
)

func IsUptoDate(s v1alpha1.ApplicationProfileParameters, t models.ApplicationProfileAttributes) bool {
	observed := &v1alpha1.ApplicationProfileParameters{
		Tenant:    s.Tenant,
		NameAlias: t.NameAlias,
	}
	return cmp.Equal(observed, &s)
}
