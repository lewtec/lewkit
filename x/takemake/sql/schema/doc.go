// Package schema is the catalog after a SQL log runs.
//
// [New] returns an empty [Database]. [Database.Apply] folds a
// [github.com/lewtec/lewkit/x/takemake/sql.File] in order. [Database.Make]
// returns the CreateTable snapshot of that catalog. The statement
// language does not import this package. The contract is in DESIGN.md
// beside x/takemake. This scaffold returns
// [github.com/lewtec/lewkit/x/errs.ErrScaffold].
package schema
