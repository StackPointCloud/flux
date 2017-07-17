package helm

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/weaveworks/flux"
	"github.com/weaveworks/flux/cluster"
)

// func TestHelmClusterInterface(t *testing.T) {
// 	var helm cluster.Cluster
// 	helm, err := NewHelmCLI()
// 	if err != nil {
// 		t.Errorf("Helm creation failed, %s", err)
// 	}
// 	_, ok := helm.(cluster.Cluster)
// 	if !ok {
// 		t.Error("cannot cast Helm to Cluster")
// 	}
// }

func TestHelmPing(t *testing.T) {
	var helm cluster.Cluster
	helm, err := NewHelmCLI()
	if err != nil {
		t.Errorf("Helm creation failed, %s", err)
	}
	err = helm.Ping()
	if err != nil {
		t.Errorf("Helm Ping failed, %s", err)
	}
}

func TestHelmParseListLine(t *testing.T) {
	line := "promop	1       	Tue Jul 11 23:05:37 2017	 DEPLOYED	prometheus-operator-0.0.6	monitoring"
	release, err := parseReleaseResponseLine(line)
	assert.Nil(t, err)
	assert.Equal(t, "promop", release.name)
	assert.Equal(t, "DEPLOYED", release.status)
	assert.Equal(t, "prometheus-operator", release.chart)
	assert.Equal(t, "0.0.6", release.version)
	assert.Equal(t, "monitoring", release.namespace)

	line = "pk8s  	1       	Wed Jul 12 12:39:42 2017	DEPLOYED	prometheus-0.1.3         	monitoring"
	release, err = parseReleaseResponseLine(line)
	assert.Nil(t, err)
	assert.Equal(t, "pk8s", release.name)
	assert.Equal(t, "DEPLOYED", release.status)
	assert.Equal(t, "prometheus", release.chart)
	assert.Equal(t, "0.1.3", release.version)
	assert.Equal(t, "monitoring", release.namespace)
}
func TestHelmAllServices(t *testing.T) {
	var helm cluster.Cluster
	helm, err := NewHelmCLI()
	if err != nil {
		t.Errorf("Helm creation failed, %s", err)
	}
	services, err := helm.AllServices("")
	if err != nil {
		t.Errorf("Helm AllServices failed, %s", err)
	}
	if len(services) == 0 {
		t.Errorf("expected services")
	}

	for _, s := range services {
		fmt.Printf("%s %s\n", s.ID, s.Status)
	}

}

func TestHelmSomeServices(t *testing.T) {
	var helm cluster.Cluster
	helm, err := NewHelmCLI()
	if err != nil {
		t.Errorf("Helm creation failed, %s", err)
	}
	id := flux.MakeServiceID("monitoring", "promop")
	services, err := helm.SomeServices([]flux.ServiceID{id})
	if err != nil {
		t.Errorf("Helm SomeServices failed, %s", err)
	}
	if len(services) == 0 {
		t.Errorf("expected services")
	}

	for _, s := range services {
		fmt.Printf("%s %s\n", s.ID, s.Status)
	}

}
