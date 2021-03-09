package anexia

import (
	"context"
	anexia "github.com/anexia-it/go-anxcloud/pkg"
	"github.com/anexia-it/go-anxcloud/pkg/clouddns"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/provider"
)
import "github.com/anexia-it/go-anxcloud/pkg/client"

var _ provider.Provider = (*AnexiaProvider)(nil)

type AnexiaProvider struct {
	*provider.BaseProvider
	Client clouddns.API
}

func (anx *AnexiaProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	panic("implement me")
}

func (anx *AnexiaProvider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	panic("implement me")
}

func NewAnexiaProvider() (*AnexiaProvider, error) {
	anxClient, err := client.New(client.AuthFromEnv(false))
	if err != nil {
		return nil, err
	}

	provider := &AnexiaProvider{
		Client: anexia.NewAPI(anxClient).CloudDNS(),
	}

	return provider, nil
}


