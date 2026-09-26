package mac

import (
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
func Create(opts Options) error {
	cfg, err := opts.Config.withDefaults()
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
