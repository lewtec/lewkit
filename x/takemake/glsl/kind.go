package glsl

// Stage is vertex, fragment, or compute.
// The zero value is unknown. [Take] infers a stage from the text.
//
//lewkit:enum invalid=unknown
type Stage uint8

const (
	Vertex Stage = iota + 1
	Fragment
	Compute
)

// Profile is the #version profile.
// The zero value prints nothing.
//
//lewkit:enum invalid=none
type Profile uint8

const (
	Core Profile = iota + 1
	Compatibility
	ES
)

// Storage is the interface keyword on a declaration.
// The zero value prints nothing.
//
//lewkit:enum invalid=none
type Storage uint8

const (
	StorageConst Storage = iota + 1
	StorageIn
	StorageOut
	StorageInOut
	StorageUniform
	StorageBuffer
	StorageShared
)

// Kind is the shape of a literal.
//
//lewkit:enum invalid=unknown
type Kind uint8

const (
	KindBool Kind = iota + 1
	KindInt
	KindUint
	KindFloat
	KindDouble
)

// Rank is a precision qualifier.
// The zero value prints nothing.
//
//lewkit:enum invalid=none
type Rank uint8

const (
	Lowp Rank = iota + 1
	Mediump
	Highp
)

// Interp is an interpolation qualifier.
// The zero value prints nothing.
//
//lewkit:enum invalid=none
type Interp uint8

const (
	Smooth Interp = iota + 1
	Flat
	NoPerspective
)

// Aux is a storage auxiliary qualifier.
// The zero value prints nothing.
//
//lewkit:enum invalid=none
type Aux uint8

const (
	Centroid Aux = iota + 1
	Sample
	Patch
)

// Memory is a memory qualifier. A declaration may hold several.
//
//lewkit:enum invalid=none
type Memory uint8

const (
	Coherent Memory = iota + 1
	Volatile
	Restrict
	Readonly
	Writeonly
)

// Behavior is the action on an #extension line.
//
//lewkit:enum invalid=unknown
type Behavior uint8

const (
	Enable Behavior = iota + 1
	Require
	Warn
	Disable
)
