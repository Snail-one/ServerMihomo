package features

import "testing"

func TestDefaultRegistryOrderAndIDs(t *testing.T) {
	registry := Default()
	features := registry.Features()
	wantIDs := []string{"install", "subscription", "service", "uninstall"}
	if len(features) != len(wantIDs) {
		t.Fatalf("len(features) = %d, want %d", len(features), len(wantIDs))
	}

	seen := make(map[string]struct{}, len(features))
	for i, feature := range features {
		if feature.ID() != wantIDs[i] {
			t.Fatalf("features[%d].ID() = %q, want %q", i, feature.ID(), wantIDs[i])
		}
		if _, ok := seen[feature.ID()]; ok {
			t.Fatalf("duplicate feature ID: %s", feature.ID())
		}
		seen[feature.ID()] = struct{}{}
	}
}
