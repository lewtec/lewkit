package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// BashCompleteLine is the bash registration for complete -C.
// Bash does not discover this. Eval the line, or drop it in
// bash-completion's completions directory. Zsh needs
// `autoload -U +X bashcompinit && bashcompinit` first.
func BashCompleteLine(name string) string {
	quoted := bashSingleQuote(name)
	return "complete -C " + quoted + " " + quoted
}

func bashSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func wordsFromCompletionEnv(getenv func(string) string) ([]string, bool) {
	line := getenv("COMP_LINE")
	if line == "" {
		return nil, false
	}
	if point := getenv("COMP_POINT"); point != "" {
		n, err := strconv.Atoi(point)
		if err == nil {
			if n < 0 {
				n = 0
			}
			if n > len(line) {
				n = len(line)
			}
			line = line[:n]
		}
	}
	return wordsAfterCommand(line), true
}

func wordsAfterCommand(line string) []string {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return []string{""}
	}
	args := append([]string(nil), fields[1:]...)
	if hasTrailingSpace(line) {
		args = append(args, "")
	}
	return args
}

func hasTrailingSpace(line string) bool {
	r, size := utf8.DecodeLastRuneInString(line)
	if size == 0 {
		return false
	}
	return unicode.IsSpace(r)
}

func writeEnvCompletions[T any](w io.Writer, getenv func(string) string) bool {
	words, ok := wordsFromCompletionEnv(getenv)
	if !ok {
		return false
	}
	suggestions, err := Complete[T](words...)
	if err != nil {
		return true
	}
	for _, suggestion := range suggestions {
		fmt.Fprintln(w, suggestion.Text)
	}
	return true
}

func exitIfCompleting[T any]() {
	if writeEnvCompletions[T](os.Stdout, os.Getenv) {
		os.Exit(0)
	}
}
