// Package sql is one SQL statement log.
//
// [NewFile] returns an empty [File]. Methods return the receiver and
// store the child: CreateTable, AlterTable, DropTable, CreateIndex,
// Select, Insert, Update, and Delete. [Take] parses the log. [Make]
// prints it. The catalog after the log runs is x/takemake/sql/schema.
// This package does not import that catalog. The contract is in
// DESIGN.md beside the parent directory. This scaffold returns
// [github.com/lewtec/lewkit/x/errs.ErrScaffold].
package sql
