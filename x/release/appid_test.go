package release

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateAppID(t *testing.T) {
	require.NoError(t, ValidateAppID("br.tec.lew.myapp"))

	require.ErrorIs(t, ValidateAppID(""), ErrAppIDRequired)
	require.ErrorIs(t, ValidateAppID("not a domain"), ErrAppIDNotReverseDNS)
	require.ErrorIs(t, ValidateAppID(`br.tec/lew`), ErrAppIDPathChars)
	require.ErrorIs(t, ValidateAppID(strings.Repeat("ab.", 80)+"c"), ErrAppIDTooLong)
}

func TestAppIDEnv(t *testing.T) {
	t.Setenv("LEWKIT_APP_ID", "br.tec.lew.env")
	previous := appID
	appID = ""
	t.Cleanup(func() { appID = previous })
	id, err := AppID()
	require.NoError(t, err)
	require.Equal(t, "br.tec.lew.env", id)
}

func TestAppIDStampWins(t *testing.T) {
	t.Setenv("LEWKIT_APP_ID", "br.tec.lew.env")
	previous := appID
	appID = "br.tec.lew.stamp"
	t.Cleanup(func() { appID = previous })
	id, err := AppID()
	require.NoError(t, err)
	require.Equal(t, "br.tec.lew.stamp", id)
}

func TestAppIDMissing(t *testing.T) {
	t.Setenv("LEWKIT_APP_ID", "")
	previous := appID
	appID = ""
	t.Cleanup(func() { appID = previous })
	_, err := AppID()
	require.ErrorIs(t, err, ErrAppIDRequired)
	require.True(t, errors.Is(err, ErrAppIDRequired))
}
