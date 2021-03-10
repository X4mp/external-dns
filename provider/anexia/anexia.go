package anexia

import (
	"context"
	"fmt"
	anexia "github.com/anexia-it/go-anxcloud/pkg"
	"github.com/anexia-it/go-anxcloud/pkg/clouddns"
	"github.com/anexia-it/go-anxcloud/pkg/clouddns/zone"
	"github.com/opencontainers/runc/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/genproto/googleapis/privacy/dlp/v2"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/provider"
	"strconv"
	"strings"
)
import "github.com/anexia-it/go-anxcloud/pkg/client"

var _ provider.Provider = (*AnexiaProvider)(nil)

const (
	actionCreate = "CREATE"
	actionDelete = "DELETE"
	actionUpdate = "UPDATE"
)

type AnexiaChangeSet struct {
	Action string
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
	requestChangeSet := make([]*AnexiaChangeSet, 0, len(changes.Create) + len(changes.UpdateNew) + len(changes.Delete))
	requestChangeSet = append(requestChangeSet, anx.newChangeSet(actionCreate, changes.Create)...)
	requestChangeSet = append(requestChangeSet, anx.newChangeSet(actionUpdate, changes.UpdateNew)...)
	requestChangeSet = append(requestChangeSet, anx.newChangeSet(actionDelete, changes.Delete)...)
	return anx.applyChanges(ctx, requestChangeSet)
}

func (anx *AnexiaProvider) newChangeSet(action string, endpoints []*endpoint.Endpoint) []*AnexiaChangeSet {
	changes := make([]*AnexiaChangeSet, len(endpoints))

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
			Record:   recordRequest,
		}
		changes = append(changes, change)
	}
	return changes
}

func (anx *AnexiaProvider) applyChanges(ctx context.Context, requestChangeSet []*AnexiaChangeSet) error {
	zones, err := anx.Client.Zone().List(ctx)
	if err != nil {
		return err
	}

	zoneIDName := provider.ZoneIDName{}
	zoneRecordIDs := make(map[string]map[string]uuid.UUID)
	for id, z := range zones {
		zoneIDName.Add(strconv.Itoa(id), z.Name)
		records, err := anx.Client.Zone().ListRecords(ctx, z.Name)
		if err != nil {
			return err
		}

		if zoneRecordIDs[z.Name] == nil {
			zoneRecordIDs[z.Name] = make(map[string]uuid.UUID)
		}

		for _, r := range records {
			zoneRecordIDs[z.Name][r.Name] = r.Identifier
		}
	}



	for _, change := range requestChangeSet {
		_, zoneName := zoneIDName.FindZone(change.Record.Name)

		if change.Record.Type == endpoint.RecordTypeCNAME {
			provider.EnsureTrailingDot(change.Record.RData)
		}
		change.Record.Name = strings.Trim(strings.TrimSuffix(change.Record.Name, zoneName), ". ")

		if change.Action == actionCreate {
			_, err := anx.Client.Zone().NewRecord(ctx, zoneName, change.Record)
			if err != nil {
				return err
			}

			log.Debug("Record created successfully")
			continue
		}

		recordID := zoneRecordIDs[zoneName][change.Record.Name]
		if change.Action == actionUpdate {
			_, err := anx.Client.Zone().UpdateRecord(ctx, zoneName, recordID, change.Record)
			if err != nil {
				return err
			}

			log.Debug("Record updated successfully")
			continue
		}

		if change.Action == actionDelete {
			err := anx.Client.Zone().DeleteRecord(ctx, zoneName, recordID)
			if err != nil {
				return err
			}

			log.Debug("Record deleted successfully")
			continue
		}
	}
	return nil
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

