package profile

import (
	"context"
	"os"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProfileActuallyFills(t *testing.T) {
	temp := t.TempDir()

	ctx, cancel := context.WithTimeout(t.Context(), 1000*time.Millisecond)
	defer cancel()
	p := NewProfile(temp)
	assert.NoError(t, p.Run(ctx))
	for _, prof := range pprof.Profiles() {
		//assert.NotZero(t, prof.Count())
		filename := p.file(prof.Name())
		stat, err := os.Stat(filename)
		if assert.NoError(t, err) {
			assert.NotZero(t, stat.Size())
		}
	}
}
