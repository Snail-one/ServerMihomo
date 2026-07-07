package feature

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"snailproxy/internal/domain/mihomo"
	"snailproxy/internal/infra/platform"
	"snailproxy/internal/terminal"
)

var ErrReturn = errors.New("返回上级菜单")

type Feature interface {
	ID() string
	Label() string
	Order() int
	Run(context.Context, Runtime) error
}

type Runtime interface {
	Terminal() terminal.Terminal
	NewMihomoStore() mihomo.Store
	NewPlatformManager() (platform.Manager, error)
}

type Registry struct {
	features []Feature
}

func NewRegistry(features ...Feature) (Registry, error) {
	seen := make(map[string]struct{}, len(features))
	validated := make([]Feature, 0, len(features))
	for _, candidate := range features {
		if candidate == nil {
			return Registry{}, fmt.Errorf("feature 不能为空")
		}

		id := strings.TrimSpace(candidate.ID())
		if id == "" {
			return Registry{}, fmt.Errorf("feature ID 不能为空")
		}
		if strings.TrimSpace(candidate.Label()) == "" {
			return Registry{}, fmt.Errorf("feature %s 的 Label 不能为空", id)
		}
		if _, ok := seen[id]; ok {
			return Registry{}, fmt.Errorf("feature ID 重复: %s", id)
		}
		seen[id] = struct{}{}
		validated = append(validated, candidate)
	}

	return Registry{features: validated}, nil
}

func MustRegistry(features ...Feature) Registry {
	registry, err := NewRegistry(features...)
	if err != nil {
		panic(err)
	}
	return registry
}

func (r Registry) Features() []Feature {
	features := slices.Clone(r.features)
	slices.SortStableFunc(features, func(a Feature, b Feature) int {
		if a.Order() != b.Order() {
			return a.Order() - b.Order()
		}
		return strings.Compare(a.ID(), b.ID())
	})
	return features
}
