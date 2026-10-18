package proto

import syntax "github.com/lewtec/lewkit/x/parse/proto"

func decodeString(raw string) (string, error)  { return syntax.DecodeString(raw) }
func parseIntToken(text string) (int64, error) { return syntax.ParseIntToken(text) }
