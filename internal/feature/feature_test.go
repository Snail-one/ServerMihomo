package feature

import (
	"context"
	"strings"
	"testing"
)

type testFeature struct {
	id    string
	label string
	order int
}

func (f testFeature) ID() string {
	return f.id
}

func (f testFeature) Label() string {
	return f.label
}

func (f testFeature) Order() int {
	return f.order
}

func (f testFeature) Run(context.Context, Runtime) error {
	return nil
}

func TestRegistrySortsFeaturesByOrder(t *testing.T) {
	registry, err := NewRegistry(
		testFeature{id: "subscription", label: "订阅管理", order: 20},
		testFeature{id: "install", label: "安装与更新", order: 10},
	)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	features := registry.Features()
	if got, want := features[0].ID(), "install"; got != want {
		t.Fatalf("features[0].ID() = %q, want %q", got, want)
	}
	if got, want := features[1].ID(), "subscription"; got != want {
		t.Fatalf("features[1].ID() = %q, want %q", got, want)
	}
}

func TestRegistryRejectsDuplicateIDs(t *testing.T) {
	_, err := NewRegistry(
		testFeature{id: "install", label: "安装与更新", order: 10},
		testFeature{id: "install", label: "重复功能", order: 20},
	)
	if err == nil || !strings.Contains(err.Error(), "feature ID 重复") {
		t.Fatalf("NewRegistry() error = %v, want duplicate ID error", err)
	}
}

func TestRegistryRejectsEmptyIDs(t *testing.T) {
	_, err := NewRegistry(testFeature{id: " ", label: "安装与更新", order: 10})
	if err == nil || !strings.Contains(err.Error(), "feature ID 不能为空") {
		t.Fatalf("NewRegistry() error = %v, want empty ID error", err)
	}
}

func TestRegistryRejectsEmptyLabels(t *testing.T) {
	_, err := NewRegistry(testFeature{id: "install", label: " ", order: 10})
	if err == nil || !strings.Contains(err.Error(), "Label 不能为空") {
		t.Fatalf("NewRegistry() error = %v, want empty label error", err)
	}
}
