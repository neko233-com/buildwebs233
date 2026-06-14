package platform

import "testing"

func TestDefaultRoadmapContainsTenFeatures(t *testing.T) {
	t.Parallel()

	roadmap := DefaultRoadmap()
	if roadmap.Product != "buildwebs233" {
		t.Fatalf("unexpected product: %s", roadmap.Product)
	}
	if len(roadmap.Recommended) != 10 {
		t.Fatalf("expected 10 features, got %d", len(roadmap.Recommended))
	}
}
