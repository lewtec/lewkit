package experiments

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffi/vulkan"
)

// Compute is `lewkit experiments compute`.
type Compute struct{}

func (Compute) Description() string {
	return "dispatch a Vulkan compute smoke shader"
}

func (Compute) Run(ctx context.Context) error {
	d, err := vulkan.Open(ctx)
	if err != nil {
		return err
	}
	defer d.Close()
	buf, err := d.Buffer(4)
	if err != nil {
		return err
	}
	defer buf.Close()
	sh, err := d.Shader(vulkan.SmokeSPIRV(), 1)
	if err != nil {
		return err
	}
	defer sh.Close()
	if err := d.Run(sh, 1, 1, 1, buf); err != nil {
		return err
	}
	out := make([]byte, 4)
	if err := buf.Read(out); err != nil {
		return err
	}
	fmt.Printf("%s: smoke -> %d\n", d.Name(), binary.LittleEndian.Uint32(out))
	return nil
}
