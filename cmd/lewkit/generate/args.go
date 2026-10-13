package generate

import "github.com/lewtec/lewkit/x/cmd"

func argValues(args []cmd.StringArg) []string {
	out := make([]string, len(args))
	for i := range args {
		out[i] = args[i].Value()
	}
	return out
}
