package sarif

// Level is the SARIF failure level on a result or a rule.
// The zero value is omitted from the log.
//
//lewkit:enum invalid=unset
type Level uint8

const (
	LevelNone    Level = iota + 1 // none
	LevelNote                     // note
	LevelWarning                  // warning
	LevelError                    // error
)

// Kind is why a result was produced.
// The zero value is omitted. SARIF's own default for an absent kind is
// fail; this model keeps absence and an explicit fail apart.
//
//lewkit:enum invalid=unset
type Kind uint8

const (
	KindNotApplicable Kind = iota + 1 // notApplicable
	KindPass                          // pass
	KindFail                          // fail
	KindReview                        // review
	KindOpen                          // open
)

// ColumnKind selects the unit of a column number on a run.
// The zero value is omitted.
//
//lewkit:enum invalid=unset
type ColumnKind uint8

const (
	ColumnKindUTF16   ColumnKind = iota + 1 // utf16CodeUnits
	ColumnKindUnicode                       // unicodeCodePoints
)
