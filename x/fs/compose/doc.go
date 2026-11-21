// Package compose builds a destination tree and encodes it as an [io/fs.FS].
//
// [Tree.Add] and [Tree.Merge] unify declarations. One path has one type.
// [TypeLines] accumulates keyed slots. The same key must carry the same slot.
// Structured maps merge by key. Lists are atomic: they stay only when both
// sides are deeply equal. Disagreement is an error.
//
// [Squash] lowers a source filesystem. A directory whose name ends in
// .d.tmpl becomes one [TypeLines] file at the path with that suffix removed.
// Each child file is a text slot keyed by its name. A regular file becomes
// a [TypeRef] slot, opened later against the base filesystem passed to
// [Tree.FS]. A symlink becomes a [TypeLink] slot whose body is the link
// target. Names that still end in .tmpl are skipped. Render templates
// before [Squash].
//
// [Tree.All] yields the declarations in path order. [Slot.Text], [Slot.Ref],
// and [Slot.Link] read one component. [FS.Open] renders the vector. A link
// opens as its target string, and [FS.Stat] reports [io/fs.ModeSymlink].
//
// [Register] adds a structured format under a type name. json, toml, yaml,
// ini, and xml are already registered. [Mount] constrains #StructuredType to
// the names registered at that call. [Parse] reads a #Tree value.
// [FS.Open] returns the combined file. Slot keys are not filesystem names.
package compose
