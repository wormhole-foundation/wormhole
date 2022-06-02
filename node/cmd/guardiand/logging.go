package guardiand

import (
	"unicode"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

type consoleEncoder struct {
	zapcore.Encoder
}

func (e consoleEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	buf, err := e.Encoder.EncodeEntry(entry, fields)
	if err != nil {
		buf.Free()
		return nil, err
	}

	b := buf.Bytes()
	for i := range b {
		if unicode.IsControl(rune(b[i])) && !unicode.IsSpace(rune(b[i])) {
			b[i] = '\x1A' // Substitute character
		}
	}

	return buf, nil
}
