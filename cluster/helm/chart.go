package helm

import (
	"fmt"

	"github.com/weaveworks/flux"
	"github.com/weaveworks/flux/policy"
	"github.com/weaveworks/flux/resource"
)

type Chart struct{}

func NewChart() (*Chart, error) {
	return &Chart{}, nil
}

// LoadManifests
func (c *Chart) LoadManifests(paths ...string) (map[string]resource.Resource, error) {
	return nil, fmt.Errorf("not implemented")
}

// ParseManifests
func (c *Chart) ParseManifests(allDefs []byte) (map[string]resource.Resource, error) {
	return nil, fmt.Errorf("not implemented")
}

// UpdateDefinition
func (c *Chart) UpdateDefinition(def []byte, container string, image flux.ImageID) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

// FindDefinedServices
func (c *Chart) FindDefinedServices(path string) (map[flux.ServiceID][]string, error) {
	return nil, fmt.Errorf("not implemented")
}

// UpdatePolicies modifies a manifest to apply the policy update specified
func (c *Chart) UpdatePolicies([]byte, policy.Update) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

// ServicesWithPolicy finds the services which have a particular policy set on them.
func (c *Chart) ServicesWithPolicy(path string, p policy.Policy) (policy.ServiceMap, error) {
	return nil, fmt.Errorf("not implemented")
}
