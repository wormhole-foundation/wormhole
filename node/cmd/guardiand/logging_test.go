package guardiand

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
	"unicode"
)

func TestEncodeEntry(t *testing.T) {
	var bad_buf bytes.Buffer
	var good_buf bytes.Buffer

	bad_buf.WriteString("foo\nbar")
	good_buf.WriteString("foo\x1Abar")

	b := bad_buf.Bytes()
	for i := range b {
		if unicode.IsControl(rune(b[i])) && !unicode.IsSpace(rune(b[i])) {
			b[i] = '\x1A' // Substitute character
		}
	}

	assert.Equal(t, good_buf, bad_buf)
}
