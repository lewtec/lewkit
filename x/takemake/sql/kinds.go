package sql

// Dialect is the SQL dialect of one file.
// The catalog records the same value because types differ.
//
//lewkit:enum fold
type Dialect uint8

const (
	// SQLite is the dialect of the decomp migrations.
	SQLite Dialect = iota + 1
	// Postgres is the other dialect lewkit generates.
	Postgres
)

// Action is a foreign-key referential action.
//
//lewkit:enum fold
type Action uint8

const (
	// Cascade deletes or updates the referencing rows.
	Cascade Action = iota + 1
	// Restrict rejects the change while referencing rows exist.
	Restrict
	// SetNull sets the referencing columns to null.
	SetNull
	// SetDefault sets the referencing columns to their default.
	SetDefault
	// NoAction leaves the check to the end of the statement.
	NoAction
)

// SQL is the ON DELETE or ON UPDATE clause for the action.
func (a Action) SQL() string {
	switch a {
	case Cascade:
		return "CASCADE"
	case Restrict:
		return "RESTRICT"
	case SetNull:
		return "SET NULL"
	case SetDefault:
		return "SET DEFAULT"
	case NoAction:
		return "NO ACTION"
	default:
		return ""
	}
}

// Type is a column type.
//
//lewkit:enum fold
type Type uint8

const (
	// TypeInteger is INTEGER.
	TypeInteger Type = iota + 1
	// TypeText is TEXT.
	TypeText
	// TypeBlob is BLOB.
	TypeBlob
	// TypeReal is REAL.
	TypeReal
	// TypeNumeric is NUMERIC.
	TypeNumeric
	// TypeBigInt is BIGINT.
	TypeBigInt
	// TypeBoolean is BOOLEAN.
	TypeBoolean
	// TypeTimestamp is TIMESTAMP.
	TypeTimestamp
)

// Integer is an INTEGER column.
func Integer() Type { return TypeInteger }

// Text is a TEXT column.
func Text() Type { return TypeText }

// Blob is a BLOB column.
func Blob() Type { return TypeBlob }

// Real is a REAL column.
func Real() Type { return TypeReal }

// Numeric is a NUMERIC column.
func Numeric() Type { return TypeNumeric }

// BigInt is a BIGINT column.
func BigInt() Type { return TypeBigInt }

// Boolean is a BOOLEAN column.
func Boolean() Type { return TypeBoolean }

// Timestamp is a TIMESTAMP column.
func Timestamp() Type { return TypeTimestamp }

// SQL is the type name printed in a statement.
func (t Type) SQL() string {
	switch t {
	case TypeInteger:
		return "INTEGER"
	case TypeText:
		return "TEXT"
	case TypeBlob:
		return "BLOB"
	case TypeReal:
		return "REAL"
	case TypeNumeric:
		return "NUMERIC"
	case TypeBigInt:
		return "BIGINT"
	case TypeBoolean:
		return "BOOLEAN"
	case TypeTimestamp:
		return "TIMESTAMP"
	default:
		return ""
	}
}

// Kind is one statement in a SQL log.
//
//lewkit:enum fold
type Kind uint8

const (
	// KindCreateTable is CREATE TABLE.
	KindCreateTable Kind = iota + 1
	// KindAlterTable is ALTER TABLE.
	KindAlterTable
	// KindDropTable is DROP TABLE.
	KindDropTable
	// KindCreateIndex is CREATE INDEX.
	KindCreateIndex
	// KindSelect is SELECT.
	KindSelect
	// KindInsert is INSERT.
	KindInsert
	// KindUpdate is UPDATE.
	KindUpdate
	// KindDelete is DELETE.
	KindDelete
)

// ExprKind is the form of one expression.
//
//lewkit:enum fold
type ExprKind uint8

const (
	// ExprKindIdent is a name, possibly qualified.
	ExprKindIdent ExprKind = iota + 1
	// ExprKindString is a quoted string.
	ExprKindString
	// ExprKindNumber is a numeric literal.
	ExprKindNumber
	// ExprKindPlaceholder is ? or $n.
	ExprKindPlaceholder
	// ExprKindStar is *.
	ExprKindStar
	// ExprKindCall is a function call.
	ExprKindCall
	// ExprKindUnary is a prefix operator.
	ExprKindUnary
	// ExprKindBinary is an infix operator.
	ExprKindBinary
)

// AlterKind is the change an [AlterTable] applies.
//
//lewkit:enum fold
type AlterKind uint8

const (
	// AlterKindAdd adds a column.
	AlterKindAdd AlterKind = iota + 1
	// AlterKindDrop drops a column.
	AlterKindDrop
	// AlterKindType retypes a column.
	AlterKindType
	// AlterKindRename renames the table.
	AlterKindRename
	// AlterKindRenameColumn renames a column.
	AlterKindRenameColumn
)
