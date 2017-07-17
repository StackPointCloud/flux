package helm

import (
	"testing"

	"github.com/weaveworks/flux/cluster"
)

func TestChartManifestInterface(t *testing.T) {
	var chart cluster.Manifests
	chart, err := NewChart()
	if err != nil {
		t.Errorf("Chart creation failed %s", err)
	}
	_, ok := chart.(cluster.Manifests)
	if !ok {
		t.Error("cannot cast Chart to Manifests")
	}
}
