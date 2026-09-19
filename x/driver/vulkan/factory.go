package vulkan

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"github.com/lewtec/lewkit/x/driver"
	ffivulkan "github.com/lewtec/lewkit/x/ffi/vulkan"
)

type factory struct{}

func (factory) ID() string { return "vulkan" }

func (factory) Name() string {
	listed.Lock()
	defer listed.Unlock()
	if len(listed.infos) == 1 {
		if listed.infos[0].Name != "" {
			return listed.infos[0].Name
		}
	}
	return "Vulkan"
}

func (factory) Weight() int { return 50 }

var listed struct {
	sync.Mutex
	infos []ffivulkan.Info
	err   error
	done  bool
}

func (factory) CheckCompatibility(ctx context.Context) error {
	listed.Lock()
	defer listed.Unlock()
	if listed.done {
		slog.Debug("vulkan list reuse", "count", len(listed.infos), "err", listed.err)
		return listed.err
	}
	listed.done = true
	slog.Debug("vulkan list")
	listed.infos, listed.err = ffivulkan.List(ctx)
	if listed.err != nil {
		slog.Debug("vulkan list failed", "err", listed.err)
		return listed.err
	}
	for _, info := range listed.infos {
		slog.Debug("vulkan list device", "index", info.Index, "name", info.Name, "vendor", info.Vendor, "type", info.Type)
	}
	return nil
}

func (f factory) Offers(ctx context.Context) ([]driver.Offer[Device], error) {
	if err := f.CheckCompatibility(ctx); err != nil {
		return nil, err
	}
	listed.Lock()
	infos := slices.Clone(listed.infos)
	listed.Unlock()
	offers := make([]driver.Offer[Device], 0, len(infos))
	for _, info := range infos {
		info := info
		offers = append(offers, driver.Offer[Device]{
			ID:     offerID(infos, info.Index),
			Name:   info.Name,
			Weight: offerWeight(info),
			New: func(ctx context.Context) (Device, error) {
				return openIndex(ctx, info.Index)
			},
		})
	}
	return offers, nil
}

func (factory) New(ctx context.Context) (Device, error) {
	return openIndex(ctx, 0)
}

func openIndex(ctx context.Context, index int) (Device, error) {
	slog.Debug("vulkan open", "index", index)
	native, err := ffivulkan.OpenIndex(ctx, index)
	if err != nil {
		return nil, err
	}
	slog.Debug("vulkan open ok", "index", index, "device", native.Name(), "vendor", native.Vendor(), "type", native.Type())
	return wrap(native), nil
}

func offerID(infos []ffivulkan.Info, index int) string {
	vendor := infos[index].Vendor
	if vendor == "" {
		vendor = "unknown"
	}
	deviceType := infos[index].Type
	n := 0
	typeIndex := 0
	for i, info := range infos {
		if info.Vendor != vendor || info.Type != deviceType {
			continue
		}
		if i == index {
			typeIndex = n
		}
		n++
	}
	id := "vulkan:" + vendor + ":" + deviceType.String()
	if n > 1 {
		return fmt.Sprintf("%s:%d", id, typeIndex)
	}
	return id
}

func offerWeight(info ffivulkan.Info) int {
	return info.Type.Weight()
}
