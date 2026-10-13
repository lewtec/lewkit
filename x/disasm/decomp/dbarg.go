package decomp

import "github.com/lewtec/lewkit/x/disasm/decomp/schema"

// DBArg is the --project flag. Open migrates the sqlite file and binds sqlc queries.
type DBArg = schema.DBArg

// Queries is the sqlc query set for one program database.
type Queries = schema.Queries
