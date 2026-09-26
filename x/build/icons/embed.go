package icons

import _ "embed"

//go:generate go run genassets.go

//go:embed default/lockup.png
var DefaultLockupPNG []byte
