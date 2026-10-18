package mac

import (
	"context"
	"embed"
	"fmt"

	"github.com/lewtec/lewkit/x/build/gen/common"
)

//go:embed all:template
var templateFS embed.FS

// Options controls Create.
type Options struct {
	OutDir string
	Force  bool
	Config Config
}

type templateData struct {
	Config
	Product            string
	AppNameXML         string
	VersionXML         string
	CodeString         string
	PlistURLTypes      string
	PlistDocumentTypes string
}

// Create writes an ephemeral XcodeGen host under opts.OutDir.
func Create(ctx context.Context, opts Options) error {
	cfg, err := opts.Config.withDefaults(ctx)
	if err != nil {
		return err
	}
	data := templateData{
		Config:             cfg,
		Product:            cfg.ProductName(),
		AppNameXML:         common.XMLEscape(cfg.AppName),
		VersionXML:         common.XMLEscape(cfg.VersionName),
		CodeString:         fmt.Sprintf("%d", cfg.VersionCode),
		PlistURLTypes:      cfg.Capabilities.PlistURLTypes(cfg.PackageID),
		PlistDocumentTypes: cfg.Capabilities.PlistDocumentTypes(),
	}
	raw, err := encodeConfigJSON(cfg)
	if err != nil {
		return err
	}
	return common.MaterializeHost(templateFS, data, opts.OutDir, opts.Force, raw)
}

// infoPlist is the bundle Info.plist. cfg already has defaults applied.
func infoPlist(cfg Config) ([]byte, error) {
	data := templateData{
		Config:             cfg,
		Product:            cfg.ProductName(),
		AppNameXML:         common.XMLEscape(cfg.AppName),
		VersionXML:         common.XMLEscape(cfg.VersionName),
		CodeString:         fmt.Sprintf("%d", cfg.VersionCode),
		PlistURLTypes:      cfg.Capabilities.PlistURLTypes(cfg.PackageID),
		PlistDocumentTypes: cfg.Capabilities.PlistDocumentTypes(),
	}
	raw, err := templateFS.ReadFile("template/Info.plist.tmpl")
	if err != nil {
		return nil, err
	}
	_, body, err := common.RenderFile("Info.plist.tmpl", raw, data)
	return body, err
}
