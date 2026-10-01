//go:build android && cgo

package android

import (
	"context"
	"errors"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	host "github.com/lewtec/lewkit/x/driver/android"
	"github.com/lewtec/lewkit/x/driver/battery"
	"github.com/lewtec/lewkit/x/ffi/jni"
	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

func init() { driver.Register[battery.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "battery_android" }
func (factory) Name() string { return "Android battery" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	n, err := androidffi.JavaVMs()
	if err != nil || n < 1 {
		return fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (battery.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) BatteryStatus(ctx context.Context) (battery.Status, error) {
	if err := ctx.Err(); err != nil {
		return battery.Unknown, err
	}
	known, err := readLevels()
	if err != nil {
		return battery.Unknown, err
	}
	intent, release, err := openBattery(ctx)
	if err != nil {
		return battery.Unknown, err
	}
	defer release()
	statusKey, err := host.Text(jni.StaticField("android.os.BatteryManager", "EXTRA_STATUS"))
	if err != nil {
		return battery.Unknown, err
	}
	code, err := host.Int(intent.Call("getIntExtra", statusKey, known.unknown))
	if err != nil {
		return battery.Unknown, err
	}
	return statusFrom(true, code, known)
}

func (backend) BatteryLevel(ctx context.Context) (int, error) {
	intent, release, err := openBattery(ctx)
	if err != nil {
		return 0, err
	}
	defer release()
	levelKey, err := host.Text(jni.StaticField("android.os.BatteryManager", "EXTRA_LEVEL"))
	if err != nil {
		return 0, err
	}
	scaleKey, err := host.Text(jni.StaticField("android.os.BatteryManager", "EXTRA_SCALE"))
	if err != nil {
		return 0, err
	}
	level, err := host.Int(intent.Call("getIntExtra", levelKey, -1))
	if err != nil {
		return 0, err
	}
	scale, err := host.Int(intent.Call("getIntExtra", scaleKey, -1))
	if err != nil {
		return 0, err
	}
	return percent(level, scale)
}

// openBattery returns the sticky battery intent when a battery is present.
// release frees the intent and the application context.
func openBattery(ctx context.Context) (*jni.Ref, func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	app, err := host.Context()
	if err != nil {
		return nil, nil, err
	}
	action, err := host.Text(jni.StaticField("android.content.Intent", "ACTION_BATTERY_CHANGED"))
	if err != nil {
		app.Release()
		return nil, nil, err
	}
	filter, err := jni.New("android.content.IntentFilter", action)
	if err != nil {
		app.Release()
		return nil, nil, err
	}
	defer filter.Release()
	intent, err := sticky(app, filter)
	if err != nil {
		app.Release()
		return nil, nil, err
	}
	if intent == nil {
		app.Release()
		return nil, nil, battery.ErrNoBattery
	}
	presentKey, err := host.Text(jni.StaticField("android.os.BatteryManager", "EXTRA_PRESENT"))
	if err != nil {
		intent.Release()
		app.Release()
		return nil, nil, err
	}
	present, err := host.Bool(intent.Call("getBooleanExtra", presentKey, true))
	if err != nil {
		intent.Release()
		app.Release()
		return nil, nil, err
	}
	if !present {
		intent.Release()
		app.Release()
		return nil, nil, battery.ErrNoBattery
	}
	return intent, func() {
		intent.Release()
		app.Release()
	}, nil
}

func readLevels() (levels, error) {
	var known levels
	var err error
	known.unknown, err = host.Int(jni.StaticField("android.os.BatteryManager", "BATTERY_STATUS_UNKNOWN"))
	if err != nil {
		return levels{}, err
	}
	known.charging, err = host.Int(jni.StaticField("android.os.BatteryManager", "BATTERY_STATUS_CHARGING"))
	if err != nil {
		return levels{}, err
	}
	known.discharging, err = host.Int(jni.StaticField("android.os.BatteryManager", "BATTERY_STATUS_DISCHARGING"))
	if err != nil {
		return levels{}, err
	}
	known.notCharging, err = host.Int(jni.StaticField("android.os.BatteryManager", "BATTERY_STATUS_NOT_CHARGING"))
	if err != nil {
		return levels{}, err
	}
	known.full, err = host.Int(jni.StaticField("android.os.BatteryManager", "BATTERY_STATUS_FULL"))
	if err != nil {
		return levels{}, err
	}
	return known, nil
}

// sticky reads the current battery intent. API 33 takes an export flag.
// Older platforms only have the two-argument form.
func sticky(app, filter *jni.Ref) (*jni.Ref, error) {
	flag, err := host.Int(jni.StaticField("android.content.Context", "RECEIVER_NOT_EXPORTED"))
	if err == nil {
		intent, callErr := host.Ref(app.Call("registerReceiver", nil, filter, flag))
		if callErr == nil {
			return intent, nil
		}
		if !errors.Is(callErr, jni.ErrNoMethod) {
			return nil, callErr
		}
	}
	return host.Ref(app.Call("registerReceiver", nil, filter))
}
