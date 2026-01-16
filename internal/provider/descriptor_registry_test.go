package provider

import (
	"testing"

	"github.com/felixgeelhaar/specular/internal/detect"
)

func TestRegisterProviderDescriptorAddsAndUpdates(t *testing.T) {
	originalStore, originalOrder := snapshotDescriptorState()
	defer restoreDescriptorState(originalStore, originalOrder)

	clearDescriptorRegistry()

	RegisterProviderDescriptor(ProviderDescriptor{
		Name:        "test-descriptor-a",
		Type:        ProviderTypeAPI,
		Description: "first description",
	})
	RegisterProviderDescriptor(ProviderDescriptor{
		Name:        "test-descriptor-b",
		Type:        ProviderTypeCLI,
		Description: "second description",
	})

	if got := DescriptorByName("test-descriptor-a"); got == nil {
		t.Fatal("DescriptorByName() returned nil for registered descriptor")
	}

	descList := Descriptors()
	if len(descList) != 2 {
		t.Fatalf("Descriptors() returned %d entries, want 2", len(descList))
	}
	if descList[0].Name != "test-descriptor-a" || descList[1].Name != "test-descriptor-b" {
		t.Fatalf("Descriptors() order mismatch: %#v", descList)
	}

	RegisterProviderDescriptor(ProviderDescriptor{
		Name:        "test-descriptor-a",
		Type:        ProviderTypeAPI,
		Description: "updated description",
	})

	if got := DescriptorByName("test-descriptor-a"); got == nil || got.Description != "updated description" {
		t.Fatalf("DescriptorByName() after update returned %v", got)
	}
	if len(Descriptors()) != 2 {
		t.Fatalf("Descriptors() length changed after update: got %d", len(Descriptors()))
	}
}

func TestDescriptorsReturnCopy(t *testing.T) {
	originalStore, originalOrder := snapshotDescriptorState()
	defer restoreDescriptorState(originalStore, originalOrder)

	clearDescriptorRegistry()

	RegisterProviderDescriptor(ProviderDescriptor{
		Name:        "copy-descriptor",
		Type:        ProviderTypeCLI,
		Description: "original",
	})

	descList := Descriptors()
	if len(descList) != 1 {
		t.Fatalf("Descriptors() returned %d entries, want 1", len(descList))
	}

	descList[0].Description = "mutated"
	if stored := DescriptorByName("copy-descriptor"); stored.Description != "original" {
		t.Fatalf("DescriptorByName() reflected mutation: %s", stored.Description)
	}
}

func TestAggregatesFromDetection(t *testing.T) {
	originalStore, originalOrder := snapshotDescriptorState()
	defer restoreDescriptorState(originalStore, originalOrder)

	clearDescriptorRegistry()

	RegisterProviderDescriptor(ProviderDescriptor{
		Name:        "agg-provider",
		Type:        ProviderTypeCLI,
		Description: "aggregated",
	})

	ctx := &detect.Context{
		Providers: map[string]detect.ProviderInfo{
			"agg-provider": {
				Available: true,
				EnvVar:    "DETECTED_ENV",
			},
		},
	}

	aggregates := AggregatesFromDetection(ctx)
	if len(aggregates) != 1 {
		t.Fatalf("AggregatesFromDetection returned %d entries, want 1", len(aggregates))
	}

	if !aggregates[0].Status.Available {
		t.Error("Expected aggregate to be available")
	}
	if aggregates[0].Status.VisibleReason != "DETECTED_ENV" {
		t.Errorf("Expected visible reason to be DETECTED_ENV, got %q", aggregates[0].Status.VisibleReason)
	}

	if AggregatesFromDetection(nil) != nil {
		t.Error("AggregatesFromDetection(nil) expected nil")
	}
}
