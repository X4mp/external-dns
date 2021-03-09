package anexia

import (
	"context"
	"fmt"
	anexia "github.com/anexia-it/go-anxcloud/pkg"
	"github.com/anexia-it/go-anxcloud/pkg/clouddns"
	"github.com/anexia-it/go-anxcloud/pkg/clouddns/zone"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/provider"
)
import "github.com/anexia-it/go-anxcloud/pkg/client"

var _ provider.Provider = (*AnexiaProvider)(nil)

type AnexiaChangeSet struct {
	Action string
	ZoneName string
	Record zone.RecordRequest
}

type AnexiaProvider struct {
	*provider.BaseProvider
	Client clouddns.API
}

func (anx *AnexiaProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	zones, err := anx.Client.Zone().List(ctx)
	if err != nil {
		return nil, err
	}

	endpoints := []*endpoint.Endpoint{}
	for _, zone := range zones {
		records, err := anx.Client.Zone().ListRecords(ctx, zone.Name)
		if err != nil {
			return nil, err
		}

		for _, r := range records {
			if provider.SupportedRecordType(r.Type) {
				name := fmt.Sprintf("%s.%s", r.Name, zone.Name)
				endpoints = append(endpoints, endpoint.NewEndpoint(name, r.Type, r.RData))
			}
		}
	}

	return endpoints, nil
}

func (anx *AnexiaProvider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	panic("implement me")
}

func (anx *AnexiaProvider) newChangeSet(action string, endpoints []*endpoint.Endpoint) []*AnexiaChangeSet {
	chanes := make([]*AnexiaChangeSet, len(endpoints))

	// TODO investigate ProviderSpecifics to transport region value
	for _, e := range endpoints {
		recordRequest := zone.RecordRequest{
			Name:   e.DNSName,
			Type:   e.RecordType,
			RData:  e.Targets[0],
		}
		if e.RecordTTL.IsConfigured() {
			recordRequest.TTL = int(e.RecordTTL)
		}

		change := &AnexiaChangeSet{
			Action:   action,
			Record:   zone.RecordRequest{},
		}
		chanes = append(chanes, change)
	}
	return chanes
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

