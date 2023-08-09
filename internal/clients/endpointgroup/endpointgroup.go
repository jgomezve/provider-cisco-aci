package endpointgroup

import (
	"fmt"
	"strings"

	aciclient "github.com/ciscoecosystem/aci-go-client/v2/client"
	"github.com/ciscoecosystem/aci-go-client/v2/models"
	"github.com/google/go-cmp/cmp"
	"github.com/jgomezve/provider-aci/apis/application-management/v1alpha1"
)

func IsUptoDate(a *aciclient.Client, s *v1alpha1.EndpointGroup, t models.ApplicationEPGAttributes) bool {

	bdName := ""
	dn := fmt.Sprintf("uni/tn-%s/ap-%s/epg-%s", s.Spec.ForProvider.Tenant, s.Spec.ForProvider.ApplicationProfile, s.Name)
	fvRsBdData, err := a.ReadRelationfvRsBdFromApplicationEPG(dn)
	if err == nil {
		bdName = strings.TrimPrefix(strings.Split(fvRsBdData.(string), "/")[2], "BD-")
	}

	observed := &v1alpha1.EndpointGroupParameters{
		PreferedGroup:      t.PrefGrMemb,
		BridgeDomain:       bdName,
		Tenant:             s.Spec.ForProvider.Tenant,
		ApplicationProfile: s.Spec.ForProvider.ApplicationProfile,
	}

	return cmp.Equal(observed, &s.Spec.ForProvider)
}
