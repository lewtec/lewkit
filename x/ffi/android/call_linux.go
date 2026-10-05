//go:build linux

package android

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	androidapp "github.com/AndroidGoLab/binder/android/app"
	androidmedia "github.com/AndroidGoLab/binder/android/media"
	androidos "github.com/AndroidGoLab/binder/android/os"
	androidview "github.com/AndroidGoLab/binder/android/view"
	gbinder "github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/servicemanager"
	"golang.org/x/sys/unix"

	"github.com/lewtec/lewkit/x/release"
)

const (
	adjustToggleMute      int32 = 101
	wakeReasonApplication int32 = 2
)

var (
	errServiceMissing = errors.New("service is missing")
	errSuspendRefused = errors.New("force suspend refused")
	errNightMode      = errors.New("night mode has no effective value")
)

type raw struct {
	sm *servicemanager.ServiceManager
}

func (r *raw) service(ctx context.Context, name servicemanager.ServiceName) (gbinder.IBinder, error) {
	svc, err := r.sm.GetService(ctx, name)
	if err != nil {
		return nil, err
	}
	if svc == nil {
		return nil, fmt.Errorf("%w: %s", errServiceMissing, name)
	}
	return svc, nil
}

func (r *raw) audio(ctx context.Context) (*androidmedia.AudioServiceProxy, error) {
	svc, err := r.service(ctx, servicemanager.AudioService)
	if err != nil {
		return nil, err
	}
	return androidmedia.NewAudioServiceProxy(svc), nil
}

func (r *raw) streamVolume(ctx context.Context, stream int32) (int32, int32, bool, error) {
	audio, err := r.audio(ctx)
	if err != nil {
		return 0, 0, false, err
	}
	cur, err := audio.GetStreamVolume(ctx, stream)
	if err != nil {
		return 0, 0, false, err
	}
	max, err := audio.GetStreamMaxVolume(ctx, stream)
	if err != nil {
		return 0, 0, false, err
	}
	muted, err := audio.IsStreamMute(ctx, stream)
	if err != nil {
		return 0, 0, false, err
	}
	return cur, max, muted, nil
}

func (r *raw) setStreamVolume(ctx context.Context, stream, index int32) error {
	audio, err := r.audio(ctx)
	if err != nil {
		return err
	}
	return audio.SetStreamVolume(ctx, stream, index, 0)
}

func (r *raw) toggleStreamMute(ctx context.Context, stream int32) error {
	audio, err := r.audio(ctx)
	if err != nil {
		return err
	}
	return audio.AdjustStreamVolume(ctx, stream, adjustToggleMute, 0)
}

func (r *raw) power(ctx context.Context) (*androidos.PowerManagerProxy, error) {
	return androidos.GetPowerManager(ctx, r.sm)
}

func (r *raw) interactive(ctx context.Context) (bool, error) {
	power, err := r.power(ctx)
	if err != nil {
		return false, err
	}
	return power.IsInteractive(ctx)
}

func (r *raw) wake(ctx context.Context) error {
	power, err := r.power(ctx)
	if err != nil {
		return err
	}
	return power.WakeUp(ctx, elapsedRealtime(), wakeReasonApplication, release.Name())
}

func (r *raw) sleep(ctx context.Context) error {
	power, err := r.power(ctx)
	if err != nil {
		return err
	}
	return power.GoToSleep(ctx, elapsedRealtime(), 0, 0)
}

func (r *raw) forceSuspend(ctx context.Context) error {
	power, err := r.power(ctx)
	if err != nil {
		return err
	}
	ok, err := power.ForceSuspend(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return errSuspendRefused
	}
	return nil
}

func (r *raw) reboot(ctx context.Context) error {
	power, err := r.power(ctx)
	if err != nil {
		return err
	}
	return power.Reboot(ctx, false, "", false)
}

func (r *raw) shutdown(ctx context.Context) error {
	power, err := r.power(ctx)
	if err != nil {
		return err
	}
	return power.Shutdown(ctx, false, "", false)
}

func (r *raw) lock(ctx context.Context) error {
	svc, err := r.service(ctx, servicemanager.WindowService)
	if err != nil {
		return err
	}
	wm := androidview.NewWindowManagerProxy(svc)
	return wm.LockNow(ctx, androidos.Bundle{})
}

func (r *raw) night(ctx context.Context) (bool, error) {
	if dark, ok, err := r.nightFromConfiguration(ctx); err != nil {
		return false, err
	} else if ok {
		return dark, nil
	}
	svc, err := r.service(ctx, servicemanager.UiModeService)
	if err != nil {
		return false, err
	}
	mode, err := androidapp.NewUiModeManagerProxy(svc).GetNightMode(ctx)
	if err != nil {
		return false, err
	}
	dark, ok := NightFromSetting(mode)
	if !ok {
		return false, fmt.Errorf("%w: %d", errNightMode, mode)
	}
	return dark, nil
}

func (r *raw) nightFromConfiguration(ctx context.Context) (dark bool, ok bool, err error) {
	svc, err := r.service(ctx, servicemanager.ActivityService)
	if err != nil {
		return false, false, nil
	}
	cfg, err := androidapp.NewActivityManagerProxy(svc).GetConfiguration(ctx)
	if err != nil {
		return false, false, nil
	}
	dark, ok = NightFromUIMode(cfg.UiMode)
	return dark, ok, nil
}

func (r *raw) dataDir(ctx context.Context, packageName string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return dataDirFor(packageName, os.Getuid(), dirExists)
}

func elapsedRealtime() int64 {
	var ts unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_BOOTTIME, &ts); err != nil {
		return time.Now().UnixMilli()
	}
	// Sec is int32 on 32-bit Linux. Widen it before multiplying.
	return int64(ts.Sec)*1000 + int64(ts.Nsec)/1e6
}
