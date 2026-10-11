package vulkan

import (
	"context"
	"fmt"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
	drvvulkan "github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
)

func init() {
	var zero drvvulkan.DeviceType
	for _, kind := range zero.Values() {
		driver.Register[ndarray.Evaluator](kindFactory{kind: kind.String()})
	}
}

// kindFactory is ndeval_vulkan:<type>. Weight 0 keeps ndarray.Open on
// gpuFactory (weight 50). The ndeval expression opens this id itself.
type kindFactory struct{ kind string }

func (f kindFactory) ID() string   { return "ndeval_vulkan:" + f.kind }
func (f kindFactory) Name() string { return "Vulkan " + f.kind }
func (kindFactory) Weight() int    { return 0 }

func (f kindFactory) CheckCompatibility(ctx context.Context) error {
	handles, err := drvvulkan.List(ctx)
	if err != nil {
		return err
	}
	_, err = pickDevice(handles, f.kind)
	return err
}

func (f kindFactory) New(ctx context.Context) (ndarray.Evaluator, error) {
	handles, err := drvvulkan.List(ctx)
	if err != nil {
		return nil, err
	}
	handle, err := pickDevice(handles, f.kind)
	if err != nil {
		return nil, err
	}
	device, err := handle.Open(ctx)
	if err != nil {
		return nil, err
	}
	return &gpuEvaluator{device: device, own: true}, nil
}

func pickDevice(handles []driver.Handle[drvvulkan.Device], kind string) (driver.Handle[drvvulkan.Device], error) {
	want, err := drvvulkan.ParseDeviceType(kind)
	if err != nil {
		return driver.Handle[drvvulkan.Device]{}, err
	}
	for _, handle := range handles {
		if deviceKind(handle.ID) == want {
			return handle, nil
		}
	}
	if want == drvvulkan.DeviceTypeIntegrated {
		for _, handle := range handles {
			if deviceKind(handle.ID) == drvvulkan.DeviceTypeSoftware {
				continue
			}
			return handle, nil
		}
	}
	return driver.Handle[drvvulkan.Device]{}, fmt.Errorf("no %s vulkan device", kind)
}

func deviceKind(id string) drvvulkan.DeviceType {
	parts := strings.Split(id, ":")
	if len(parts) < 3 {
		return drvvulkan.DeviceTypeOther
	}
	kind, err := drvvulkan.ParseDeviceType(parts[2])
	if err != nil {
		return drvvulkan.DeviceTypeOther
	}
	return kind
}
