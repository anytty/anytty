package endpoint

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
)

func TestUpdateSerializesConcurrentRegistryMutations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "endpoints.yaml")
	const mutations = 12
	start := make(chan struct{})
	errorsByMutation := make(chan error, mutations)
	var wait sync.WaitGroup
	for index := 0; index < mutations; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			id := EndpointID(fmt.Sprintf("endpoint-%02d", index))
			_, err := Update(path, true, func(registry Registry) (Registry, error) {
				if registry.Endpoints == nil {
					registry.Endpoints = map[EndpointID]Endpoint{}
				}
				registry.Version = RegistryVersion
				registry.Endpoints[id] = NewSSHEndpoint(id, string(id), "host-"+string(id), "", "127.0.0.1:41120", "127.0.0.1:41121", ConnectOnDemand)
				if registry.Default == "" {
					registry.Default = id
				}
				return registry, nil
			})
			errorsByMutation <- err
		}(index)
	}
	close(start)
	wait.Wait()
	close(errorsByMutation)
	for err := range errorsByMutation {
		if err != nil {
			t.Fatal(err)
		}
	}
	registry, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.Endpoints) != mutations {
		t.Fatalf("concurrent registry updates lost mutations: got %d want %d", len(registry.Endpoints), mutations)
	}
}
