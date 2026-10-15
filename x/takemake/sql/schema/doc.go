// Package schema is the catalog after a SQL log runs.
//
// [New] returns an empty [Database]. [Database.Apply] folds a
// [github.com/lewtec/lewkit/x/takemake/sql.File] in order. DropTable removes
// a table. AlterTable adds, drops, renames, or retypes a column.
// [Database.Make] returns the CreateTable snapshot and then its indexes.
// Applying that snapshot to an empty catalog reproduces the tables, columns,
// keys, and foreign keys.
//
// The statement language does not import this package.
package schema
