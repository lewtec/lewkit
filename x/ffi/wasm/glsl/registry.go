package glsl

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"sync"
)

// ErrExist means [RegisterHash] already stored different SPIR-V for that hash.
var ErrExist = errors.New("shader already registered")

type sumKey struct {
	stage Stage
	sum   [32]byte
}

var (
	registryMu sync.RWMutex
	registry   = map[sumKey][]byte{}
)

// Hash is the SHA-256 of stage and src.
// The stage is a little-endian uint32 prefix so two stages of one source differ.
func Hash(stage Stage, src []byte) [32]byte {
	buf := make([]byte, 4+len(src))
	binary.LittleEndian.PutUint32(buf, uint32(stage))
	copy(buf[4:], src)
	return sha256.Sum256(buf)
}

// RegisterHash stores spirv under stage and sum.
// sum is [Hash] of the stage and the GLSL source.
// The same sum with the same SPIR-V is a no-op.
// spirv is copied.
func RegisterHash(stage Stage, sum [32]byte, spirv []byte) error {
	if !knownStage(stage) || !validSPIRV(spirv) {
		return ErrCompile
	}
	key := sumKey{stage: stage, sum: sum}
	registryMu.Lock()
	defer registryMu.Unlock()
	if old, ok := registry[key]; ok {
		if bytes.Equal(old, spirv) {
			return nil
		}
		return ErrExist
	}
	registry[key] = append([]byte(nil), spirv...)
	return nil
}

// MustRegisterHash is [RegisterHash] that panics on error.
func MustRegisterHash(stage Stage, sum [32]byte, spirv []byte) {
	if err := RegisterHash(stage, sum, spirv); err != nil {
		panic(err)
	}
}

// Lookup copies the SPIR-V registered for stage and src.
func Lookup(stage Stage, src []byte) ([]byte, bool) {
	key := sumKey{stage: stage, sum: Hash(stage, src)}
	registryMu.RLock()
	spv, ok := registry[key]
	registryMu.RUnlock()
	if !ok {
		return nil, false
	}
	return append([]byte(nil), spv...), true
}

func knownStage(stage Stage) bool {
	return stage == StageVertex || stage == StageFragment || stage == StageCompute
}

func validSPIRV(spirv []byte) bool {
	return IsSPIRV(spirv) && len(spirv) >= 20 && len(spirv)%4 == 0
}
