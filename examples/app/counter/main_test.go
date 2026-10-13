package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCounterIncDecReset(t *testing.T) {
	srv := httptest.NewServer(newCounter())
	defer srv.Close()
	client := &http.Client{Timeout: 5 * time.Second}
	post := func(op string) string {
		t.Helper()
		res, err := client.PostForm(srv.URL+"/", url.Values{"op": {op}})
		require.NoError(t, err)
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		const mark = `class="count">`
		text := string(body)
		require.Contains(t, text, mark)
		rest := text[strings.Index(text, mark)+len(mark):]
		end := strings.Index(rest, "<")
		require.GreaterOrEqual(t, end, 0)
		return rest[:end]
	}
	require.Equal(t, "1", post("inc"))
	require.Equal(t, "2", post("inc"))
	require.Equal(t, "1", post("dec"))
	require.Equal(t, "0", post("reset"))
}
