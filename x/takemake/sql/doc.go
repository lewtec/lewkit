// Package sql is a SQL statement log.
//
// [NewFile] returns an empty [File] in the sqlite dialect. Methods return
// the receiver and store the child: CreateTable, AlterTable, DropTable,
// CreateIndex, Select, Insert, Update, and Delete. [Take] parses that log.
// [Make] prints it. A composed file and a taken file are the same value shape.
//
// [File.Postgres] selects the other dialect. The catalog after the log runs
// is x/takemake/sql/schema. This package does not import that catalog.
//
// Take reads the migration and query shape used in this module: table
// definitions, alters, drops, indexes, and select, insert, update, and
// delete without joins.
package sql
