package fs

import "github.com/lewtec/lewkit/x/path"

// Keep reports whether to copy p. dir is true when p is a directory.
// A nil Keep keeps everything.
//
// A false file is skipped. A false directory is pruned: [Walk] does
// not descend ([io/fs.SkipDir]); [Filter] skips names under that
// prefix and does not pass their bodies on. Return true for a
// directory to enter it, for example so **/*.go can match children.
type Keep func(p path.Path, dir bool) bool

func (k Keep) call(p path.Path, dir bool) bool {
	if k == nil {
		return true
	}
	return k(p, dir)
}

// And keeps names that both k and other keep.
func (k Keep) And(other Keep) Keep {
	return func(p path.Path, dir bool) bool {
		return k.call(p, dir) && other.call(p, dir)
	}
}

// Or keeps names that k or other keep.
func (k Keep) Or(other Keep) Keep {
	return func(p path.Path, dir bool) bool {
		return k.call(p, dir) || other.call(p, dir)
	}
}

// Not inverts k. A nil Keep inverts to reject everything.
func (k Keep) Not() Keep {
	return func(p path.Path, dir bool) bool {
		return !k.call(p, dir)
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
