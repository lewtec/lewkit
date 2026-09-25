package linux

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/camera"
)

type backend struct{}

type cameraDevice struct {
	device string
	id     string
	name   string
}

func (backend) List(context.Context) ([]camera.Camera, error) {
	devices, err := filepath.Glob("/dev/video*")
	if err != nil {
		return nil, err
	}

	sort.Slice(devices, func(i, j int) bool {
		ii := videoIndex(devices[i])
		ij := videoIndex(devices[j])
		if ii != ij {
			return ii < ij
		}
		return devices[i] < devices[j]
	})

	cams := make([]camera.Camera, 0, len(devices))
	for _, dev := range devices {
		info, err := os.Stat(dev)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeDevice == 0 {
			continue
		}
		cams = append(cams, cameraDevice{
			device: dev,
			id:     dev,
			name:   cameraName(dev),
		})
	}
	return cams, nil
}

func (c cameraDevice) ID() string { return c.id }

func (c cameraDevice) Name() string { return c.name }

func (c cameraDevice) Capture(ctx context.Context) (image.Image, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("%w: ffmpeg not found", driver.ErrIncompatible)
	}

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-nostdin",
		"-f", "video4linux2",
		"-i", c.device,
		"-frames:v", "1",
		"-f", "image2pipe",
		"-vcodec", "png",
		"-")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return nil, fmt.Errorf("ffmpeg capture from %s failed: %w: %s", c.device, err, msg)
		}
		return nil, fmt.Errorf("ffmpeg capture from %s failed: %w", c.device, err)
	}

	img, _, err := image.Decode(bytes.NewReader(stdout.Bytes()))
	if err != nil {
		return nil, fmt.Errorf("decode camera frame from %s: %w", c.device, err)
	}
	return img, nil
}

func cameraName(device string) string {
	base := filepath.Base(device)
	sysfs := filepath.Join("/sys/class/video4linux", base, "name")
	data, err := os.ReadFile(sysfs)
	if err != nil {
		return base
	}
	name := strings.TrimSpace(string(data))
	if name == "" {
		return base
	}
	return name
}

func videoIndex(device string) int {
	base := filepath.Base(device)
	if !strings.HasPrefix(base, "video") {
		return math.MaxInt
	}
	idx, err := strconv.Atoi(strings.TrimPrefix(base, "video"))
	if err != nil {
		return math.MaxInt
	}
	return idx
}

func requireBinary(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%w: %s not found", driver.ErrIncompatible, name)
	}
	return nil
}

var _ camera.Driver = backend{}
var _ camera.Camera = cameraDevice{}
