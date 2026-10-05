package win

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"
)

// stampExe writes the app icon, PerMonitorV2 manifest, and version info
// into a windowsgui executable. The original file is replaced only after
// the patch succeeds.
func stampExe(path, icoPath string, cfg Config) error {
	ico, err := os.ReadFile(icoPath)
	if err != nil {
		return fmt.Errorf("icon: %w", err)
	}
	icon, err := winres.LoadICO(bytes.NewReader(ico))
	if err != nil {
		return fmt.Errorf("icon: %w", err)
	}
	var rs winres.ResourceSet
	if err := rs.SetIcon(winres.ID(1), icon); err != nil {
		return fmt.Errorf("icon: %w", err)
	}
	rs.SetManifest(winres.AppManifest{
		Description:         cfg.AppName,
		Compatibility:       winres.Win10AndAbove,
		DPIAwareness:        winres.DPIPerMonitorV2,
		UseCommonControlsV6: true,
		LongPathAware:       true,
	})
	var info version.Info
	info.SetFileVersion(cfg.VersionName)
	info.SetProductVersion(cfg.VersionName)
	product := cfg.ProductName()
	_ = info.Set(version.LangDefault, version.ProductName, cfg.AppName)
	_ = info.Set(version.LangDefault, version.FileDescription, cfg.AppName)
	_ = info.Set(version.LangDefault, version.InternalName, product)
	_ = info.Set(version.LangDefault, version.OriginalFilename, product+".exe")
	_ = info.Set(version.LangDefault, version.ProductVersion, cfg.VersionName)
	_ = info.Set(version.LangDefault, version.FileVersion, cfg.VersionName)
	rs.SetVersionInfo(info)

	src, err := os.Open(path)
	if err != nil {
		return err
	}
	tmp := path + ".res"
	dst, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return errors.Join(err, src.Close())
	}
	writeErr := rs.WriteToEXE(dst, src, winres.ForceCheckSum())
	if err := errors.Join(writeErr, dst.Close(), src.Close()); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("resources: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
