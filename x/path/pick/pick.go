// Package pick is a predicate on a [path.Path].
//
// The generators are [Dir] and [Match]. [File], [Glob], and [Prune]
// are derived. [And], [Or], and [Not] are the Boolean algebra.
// A nil [Pred] accepts everything.
package pick

import "github.com/lewtec/lewkit/x/path"

// Pred reports whether to pick p. dir is true when p is a directory.
// A nil Pred accepts everything.
//
// A false file is skipped. A false directory is pruned: descendants
// are not visited.
type Pred func(p path.Path, dir bool) bool

// Dir is true for directories.
var Dir Pred = func(_ path.Path, dir bool) bool { return dir }

// File is true for non-directories. File is [Not]([Dir]).
var File Pred = Not(Dir)

func call(p Pred, name path.Path, dir bool) bool {
	if p == nil {
		return true
	}
	return p(name, dir)
}

// And is true when every p is true. An empty And is true.
func And(ps ...Pred) Pred {
	return func(name path.Path, dir bool) bool {
		for _, p := range ps {
			if !call(p, name, dir) {
				return false
			}
		}
		return true
	}
}

// Or is true when any p is true. An empty Or is false.
func Or(ps ...Pred) Pred {
	return func(name path.Path, dir bool) bool {
		if len(ps) == 0 {
			return false
		}
		for _, p := range ps {
			if call(p, name, dir) {
				return true
			}
		}
		return false
	}
}

// Not inverts p. A nil Pred inverts to reject everything.
func Not(p Pred) Pred {
	return func(name path.Path, dir bool) bool {
		return !call(p, name, dir)
	}
}

// Match is true when the path matches pattern, file or directory.
func Match(pattern string) Pred {
	return func(name path.Path, _ bool) bool {
		ok, err := name.MatchGlob(pattern)
		return err == nil && ok
	}
}

// Glob picks files matching pattern and enters every directory.
// Glob is [Or]([Dir], [Match](pattern)).
func Glob(pattern string) Pred {
	return Or(Dir, Match(pattern))
}

// Prune drops directories matching pattern and their children.
// Prune is [Not]([And]([Dir], [Match](pattern))).
func Prune(pattern string) Pred {
	return Not(And(Dir, Match(pattern)))
}
