package node

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateFormatString(t *testing.T) {
	t.Parallel()

	require.Equal(t, "<host>:<port>", generateFormatString([]string{""}))
	require.Equal(t, "HTTP or HTTPS", generateFormatString([]string{"http", "https"}))
	require.Equal(t, "WS or <host>:<port> or UNIX", generateFormatString([]string{"ws", "", "unix"}))
}
