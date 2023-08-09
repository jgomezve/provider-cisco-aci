package bridgedomain

import (
	"fmt"
	"strings"

	aciclient "github.com/ciscoecosystem/aci-go-client/v2/client"
	"github.com/ciscoecosystem/aci-go-client/v2/models"
	"github.com/google/go-cmp/cmp"
	"github.com/jgomezve/provider-aci/apis/networking/v1alpha1"
)

func IsUptoDate(a *aciclient.Client, s *v1alpha1.BridgeDomain, t models.BridgeDomainAttributes) bool {

	vrfName := ""
	dn := fmt.Sprintf("uni/tn-%s/BD-%s", s.Spec.ForProvider.Tenant, s.Name)
	fvRsCtxData, err := a.ReadRelationfvRsCtxFromBridgeDomain(dn)
	if err == nil {
		vrfName = strings.TrimPrefix(strings.Split(fvRsCtxData.(string), "/")[2], "ctx-")
	}

	observed := &v1alpha1.BridgeDomainParameters{
		ArpFlood: t.ArpFlood,
		Vrf:      vrfName,
		Tenant:   s.Spec.ForProvider.Tenant,
	}

	return cmp.Equal(observed, &s.Spec.ForProvider)
}
