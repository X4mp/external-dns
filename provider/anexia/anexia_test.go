/*
Copyright 2020 The Kubernetes Authors.
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

package anexia

import (
	"context"
	"encoding/json"
	"github.com/alecthomas/assert"
	"github.com/anexia-it/go-anxcloud/pkg/client"
	"github.com/anexia-it/go-anxcloud/pkg/clouddns"
	"github.com/anexia-it/go-anxcloud/pkg/clouddns/zone"
	uuid "github.com/satori/go.uuid"
	"net/http"
	"os"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"strings"
	"testing"
	"time"
)

func TestNewAnexiaProvider(t *testing.T) {
	os.Setenv("ANEXIA_TOKEN", "myTestToken")
	anxProvider, err := NewAnexiaProvider()
	assert.NoError(t, err)
	assert.NotNil(t, anxProvider)
}

func TestAnexiaProvider_Records(t *testing.T) {
	c, server := client.NewTestClient(nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "zone.json") {
			zones := []zone.Zone{
				{
					Definition: &zone.Definition{
						Name: "anexia.test",
					},
				},
			}
			response := map[string][]zone.Zone{
				"results": zones,
			}
			err := json.NewEncoder(w).Encode(response)
			assert.NoError(t, err)
		} else {
			ttl := 300
			response := []zone.Record{
				{
					Identifier: uuid.NewV4(),
					Immutable:  false,
					Name:       "test1",
					RData:      "127.0.0.1",
					Region:     "default",
					TTL:        &ttl,
					Type:       "A",
				},
				{
					Identifier: uuid.NewV4(),
					Immutable:  false,
					Name:       "test2",
					RData:      "127.0.0.1",
					Region:     "default",
					TTL:        nil,
					Type:       "A",
				},
			}
			err := json.NewEncoder(w).Encode(response)
			assert.NoError(t, err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := AnexiaProvider{
		Client: clouddns.NewAPI(c),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*1)
	defer cancel()

	expectedEndpoints := []*endpoint.Endpoint{
		{
			DNSName:    "test1.anexia.test",
			Targets:    []string{"127.0.0.1"},
			RecordType: "A",
			RecordTTL:  300,
			Labels:     endpoint.Labels{},
		},
		{
			DNSName:    "test2.anexia.test",
			Targets:    []string{"127.0.0.1"},
			RecordType: "A",
			RecordTTL:  0,
			Labels:     endpoint.Labels{},
		},
	}
	ep, err := p.Records(ctx)
	assert.NoError(t, err)
	assert.EqualValues(t, expectedEndpoints, ep)
}

func TestAnexiaProvider_ApplyChanges(t *testing.T) {
	c, server := client.NewTestClient(nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "zone.json") {
			zones := []zone.Zone{
				{
					Definition: &zone.Definition{
						Name: "anexia.test",
					},
				},
			}
			response := map[string][]zone.Zone{
				"results": zones,
			}
			err := json.NewEncoder(w).Encode(response)
			assert.NoError(t, err)
		} else if r.Method != http.MethodGet {
			z := zone.Zone{
				Definition: &zone.Definition{
					Name: "anexia.test",
				},
			}
			err := json.NewEncoder(w).Encode(z)
			assert.NoError(t, err)
		} else {
			ttl := 300
			response := []zone.Record{
				{
					Identifier: uuid.NewV4(),
					Immutable:  false,
					Name:       "test1",
					RData:      "127.0.0.1",
					Region:     "default",
					TTL:        &ttl,
					Type:       "A",
				},
				{
					Identifier: uuid.NewV4(),
					Immutable:  false,
					Name:       "test2",
					RData:      "127.0.0.1",
					Region:     "default",
					TTL:        nil,
					Type:       "A",
				},
			}
			err := json.NewEncoder(w).Encode(response)
			assert.NoError(t, err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := AnexiaProvider{
		Client: clouddns.NewAPI(c),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*1)
	defer cancel()

	err := p.ApplyChanges(ctx, &plan.Changes{
		Create:    []*endpoint.Endpoint{
			{
				DNSName:    "test3.anexia.test",
				Targets:    []string{"127.0.0.1"},
				RecordType: "A",
				RecordTTL:  300,
				Labels:     endpoint.Labels{},
			},
		},
		UpdateNew: []*endpoint.Endpoint{
			{
				DNSName:    "test1.anexia.test",
				Targets:    []string{"10.0.0.1"},
				RecordType: "A",
				RecordTTL:  100,
				Labels:     endpoint.Labels{},
			},
		},
		Delete:    []*endpoint.Endpoint{
			{
				DNSName:    "test2.anexia.test",
				Targets:    []string{"10.0.0.1"},
				RecordType: "A",
				RecordTTL:  300,
				Labels:     endpoint.Labels{},
			},
		},
	})
	assert.NoError(t, err)
}
