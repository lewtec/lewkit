package ios

import (
	"embed"
	"fmt"

	"github.com/lewtec/lewkit/x/build/gen/common"
)

//go:embed all:template clangwrap.sh
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
	ShareProduct       string
	AppNameXML         string
	VersionXML         string
	CodeString         string
	PlistURLTypes      string
	PlistDocumentTypes string
	WakeScheme         string
	AppGroupID         string
	EmbedShare         bool
	ShareActivationXML string
}

// Create writes an ephemeral XcodeGen iOS host under opts.OutDir.
func Create(opts Options) error {
	cfg, err := opts.Config.withDefaults()
	if err != nil {
		return err
	}
	product := cfg.ProductName()
	data := templateData{
		Config:             cfg,
		Product:            product,
		ShareProduct:       product + "Share",
		AppNameXML:         common.XMLEscape(cfg.AppName),
		VersionXML:         common.XMLEscape(cfg.VersionName),
		CodeString:         fmt.Sprintf("%d", cfg.VersionCode),
		PlistURLTypes:      cfg.Capabilities.PlistURLTypes(cfg.PackageID),
		PlistDocumentTypes: cfg.Capabilities.PlistDocumentTypes(),
		WakeScheme:         cfg.Capabilities.WakeScheme(cfg.PackageID),
		AppGroupID:         common.AppGroupID(cfg.PackageID),
		EmbedShare:         cfg.Capabilities.Files != nil || cfg.Capabilities.Share != nil,
		ShareActivationXML: cfg.Capabilities.ShareActivationXML(),
	}
	raw, err := encodeConfigJSON(cfg)
	if err != nil {
		return err
	}
	return common.MaterializeHost(templateFS, data, opts.OutDir, opts.Force, raw)
}
