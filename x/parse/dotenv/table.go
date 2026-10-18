package dotenv

import xparse "github.com/lewtec/lewkit/x/parse"

// ParseTable compiles the declaration into the LR(1) table.
func ParseTable() ([]byte, error) {
	return xparse.TableBytes(declare())
}
