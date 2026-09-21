// Package compose builds a destination tree and encodes it as an [io/fs.FS].
//
// [Tree.Add] and [Tree.Merge] unify declarations. One path has one type.
// [TypeLines] accumulates keyed slots. The same key must carry the same slot.
// Structured maps merge by key. Lists are atomic: they stay only when both
// sides are deeply equal. Disagreement is an error.
//
// [Squash] lowers a source filesystem. A directory whose name ends in
// .d.tmpl becomes one [TypeLines] file at the path with that suffix removed.
// Each child file is a text slot keyed by its name. Other files become
// [TypeRef] slots, opened later against the base filesystem passed to
// [Tree.FS]. Names that still end in .tmpl are skipped. Render templates
// before [Squash].
//
// [Mount] constrains a CUE path to #Tree. [Parse] reads that value into a
// tree. [FS.Open] returns the combined file. Slot keys are not filesystem names.
package compose
