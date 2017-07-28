package helm

import (
	"fmt"
	"testing"

	"github.com/weaveworks/flux"
	"github.com/weaveworks/flux/cluster"
)

func getNewHelm(t *testing.T) cluster.Cluster {
	helm, err := NewHelmGRPC()
	if err != nil {
		t.Errorf("Helm creation failed, %s", err)
		t.FailNow()
	}
	return helm
}
func TestHelmClusterInterface(t *testing.T) {
	var helm cluster.Cluster
	helm = getNewHelm(t)
	_, ok := helm.(cluster.Cluster)
	if !ok {
		t.Error("cannot cast Helm to Cluster")
	}
}

func TestHelmPing(t *testing.T) {
	var helm cluster.Cluster
	helm = getNewHelm(t)

	err := helm.Ping()
	if err != nil {
		t.Errorf("Helm Ping failed, %s", err)
	}
}

func TestHelmAllServices(t *testing.T) {
	var helm cluster.Cluster
	helm = getNewHelm(t)

	services, err := helm.AllServices("")
	if err != nil {
		t.Errorf("Helm AllServices failed, %s", err)
	}
	if len(services) == 0 {
		t.Errorf("expected services")
	}

	for _, s := range services {
		fmt.Printf("TEST  %s %s\n", s.ID, s.Status)
	}

}

func TestHelmSomeServices(t *testing.T) {
	var helm cluster.Cluster
	helm = getNewHelm(t)

	id := flux.MakeServiceID("default", "rolling-bird-openvpn")
	services, err := helm.SomeServices([]flux.ServiceID{id})
	if err != nil {
		t.Errorf("Helm SomeServices failed, %s", err)
	}
	if len(services) == 0 {
		t.Errorf("expected services for %s", id)
	}

	for _, s := range services {
		fmt.Printf("TEST  %s %s\n", s.ID, s.Status)
	}

}
