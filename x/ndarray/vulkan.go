package ndarray

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"image"
	"log/slog"
	"slices"
	"sync"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/ffi/vulkan"
)

// Vulkan is the evaluator for a Vulkan device. Sessions are reused per kernel.
type Vulkan struct {
	Device   *vulkan.Device
	own      bool
	mu       sync.Mutex
	sessions map[*Kernel]*Session
}

var _ Evaluator = (*Vulkan)(nil)
var _ driver.DriverFactory[Evaluator] = vulkanFactory{}
var _ driver.Offerer[Evaluator] = vulkanFactory{}

func init() {
	driver.Register[Evaluator](vulkanFactory{})
}

type vulkanFactory struct{}

func (vulkanFactory) ID() string { return "ndarray_vulkan" }

func (vulkanFactory) Name() string {
	vulkanListMu.Lock()
	defer vulkanListMu.Unlock()
	if len(vulkanInfos) == 1 {
		return cmp.Or(vulkanInfos[0].Name, "Vulkan")
	}
	return "Vulkan"
}

func (vulkanFactory) Weight() int { return 50 }

var (
	vulkanListMu  sync.Mutex
	vulkanInfos   []vulkan.Info
	vulkanListErr error
	vulkanListed  bool
)

func (vulkanFactory) CheckCompatibility(ctx context.Context) error {
	vulkanListMu.Lock()
	defer vulkanListMu.Unlock()
	if vulkanListed {
		slog.Debug("vulkan list reuse", "count", len(vulkanInfos), "err", vulkanListErr)
		return vulkanListErr
	}
	vulkanListed = true
	slog.Debug("vulkan list")
	vulkanInfos, vulkanListErr = vulkan.List(ctx)
	if vulkanListErr != nil {
		slog.Debug("vulkan list failed", "err", vulkanListErr)
		return vulkanListErr
	}
	for _, info := range vulkanInfos {
		slog.Debug("vulkan list device", "index", info.Index, "name", info.Name)
	}
	return nil
}

func (f vulkanFactory) Offers(ctx context.Context) ([]driver.Offer[Evaluator], error) {
	if err := f.CheckCompatibility(ctx); err != nil {
		return nil, err
	}
	vulkanListMu.Lock()
	infos := slices.Clone(vulkanInfos)
	vulkanListMu.Unlock()
	offers := make([]driver.Offer[Evaluator], 0, len(infos))
	for _, info := range infos {
		info := info
		offers = append(offers, driver.Offer[Evaluator]{
			ID:   vulkanOfferID(info.Index, len(infos)),
			Name: info.Name,
			New: func(ctx context.Context) (Evaluator, error) {
				return openVulkanIndex(ctx, info.Index)
			},
		})
	}
	return offers, nil
}

func vulkanOfferID(index, count int) string {
	if count <= 1 {
		return "ndarray_vulkan"
	}
	return fmt.Sprintf("ndarray_vulkan:%d", index)
}

func (vulkanFactory) New(ctx context.Context) (Evaluator, error) {
	return openVulkanIndex(ctx, 0)
}

func openVulkanIndex(ctx context.Context, index int) (Evaluator, error) {
	slog.Debug("vulkan open", "index", index)
	d, err := vulkan.OpenIndex(ctx, index)
	if err != nil {
		return nil, err
	}
	slog.Debug("vulkan open ok", "index", index, "device", d.Name())
	return &Vulkan{Device: d, own: true}, nil
}

func (v *Vulkan) Run(ctx context.Context, kernel *Kernel, output []float32) error {
	session, err := v.session(ctx, kernel)
	if err != nil {
		return err
	}
	return session.Run(ctx, output)
}

func (v *Vulkan) RunRGBA(ctx context.Context, kernel *Kernel, destination *image.RGBA) error {
	session, err := v.session(ctx, kernel)
	if err != nil {
		return err
	}
	return session.RunRGBA(ctx, destination)
}

func (v *Vulkan) session(ctx context.Context, kernel *Kernel) (*Session, error) {
	if v == nil || v.Device == nil || kernel == nil {
		return nil, ErrOp
	}
	v.mu.Lock()
	if v.sessions == nil {
		v.sessions = make(map[*Kernel]*Session)
	}
	s := v.sessions[kernel]
	v.mu.Unlock()
	if s != nil {
		return s, nil
	}
	s, err := kernel.Attach(ctx, v.Device)
	if err != nil {
		return nil, err
	}
	v.mu.Lock()
	if existing := v.sessions[kernel]; existing != nil {
		v.mu.Unlock()
		if err := s.Close(); err != nil {
			return nil, err
		}
		return existing, nil
	}
	v.sessions[kernel] = s
	v.mu.Unlock()
	slog.Debug("vulkan session", "device", v.Device.Name())
	return s, nil
}

func (v *Vulkan) Close() error {
	if v == nil {
		return nil
	}
	v.mu.Lock()
	sessions := v.sessions
	v.sessions = nil
	own, d := v.own, v.Device
	v.own = false
	v.Device = nil
	v.mu.Unlock()
	var err error
	for _, s := range sessions {
		err = errors.Join(err, s.Close())
	}
	if own && d != nil {
		slog.Debug("vulkan close", "device", d.Name(), "sessions", len(sessions))
		err = errors.Join(err, d.Close())
	}
	return err
}
