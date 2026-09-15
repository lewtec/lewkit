// Package keep is a predicate on a [path.Path].
//
// [Keep] is used to filter listings. A nil Keep keeps everything.
// [Glob], [Prune], [And], [Or], and [Not] build one.
package keep

import "github.com/lewtec/lewkit/x/path"

// Keep reports whether to keep p. dir is true when p is a directory.
// A nil Keep keeps everything.
//
// A false file is skipped. A false directory is pruned: descendants
// are not visited.
type Keep func(p path.Path, dir bool) bool

func call(k Keep, p path.Path, dir bool) bool {
	if k == nil {
		return true
	}
	return k(p, dir)
}

// And keeps names that every k keeps. An empty And keeps everything.
func And(ks ...Keep) Keep {
	return func(p path.Path, dir bool) bool {
		for _, k := range ks {
			if !call(k, p, dir) {
				return false
			}
		}
		return true
	}
}

// Or keeps names that any k keeps. An empty Or keeps nothing.
func Or(ks ...Keep) Keep {
	return func(p path.Path, dir bool) bool {
		if len(ks) == 0 {
			return false
		}
		for _, k := range ks {
			if call(k, p, dir) {
				return true
			}
		}
		return false
	}
}

// Not inverts k. A nil Keep inverts to reject everything.
func Not(k Keep) Keep {
	return func(p path.Path, dir bool) bool {
		return !call(k, p, dir)
	}
}

// Glob keeps files matching pattern and enters every directory.
func Glob(pattern string) Keep {
	return func(p path.Path, dir bool) bool {
		if dir {
			return true
		}
		ok, err := p.MatchGlob(pattern)
		return err == nil && ok
	}
}

// Prune drops directories matching pattern and their children.
func Prune(pattern string) Keep {
	return func(p path.Path, dir bool) bool {
		if !dir {
			return true
		}
		ok, err := p.MatchGlob(pattern)
		return err != nil || !ok
	}
}
