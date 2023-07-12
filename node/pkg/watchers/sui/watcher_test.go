package sui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func Test_fixSuiWsURL(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		expected   string
		logMessage string
		errMessage string
	}{
		{
			name:     "valid",
			value:    "ws://1.2.3.4:5678",
			expected: "ws://1.2.3.4:5678",
		},
		{
			name:       "tilt",
			value:      "sui:9000",
			expected:   "ws://sui:9000",
			logMessage: "DEPRECATED: Prefix --suiWS address with the url scheme e.g.: ws://sui:9000 or wss://sui:9000",
		},
		{
			name:     "valid-no-port",
			value:    "ws://1.2.3.4",
			expected: "ws://1.2.3.4",
		},
		{
			name:       "no-scheme",
			value:      "1.2.3.4:5678",
			expected:   "ws://1.2.3.4:5678",
			logMessage: "DEPRECATED: Prefix --suiWS address with the url scheme e.g.: ws://1.2.3.4:5678 or wss://1.2.3.4:5678",
		},
		{
			name:       "no-scheme-no-port",
			value:      "1.2.3.4",
			expected:   "ws://1.2.3.4",
			logMessage: "DEPRECATED: Prefix --suiWS address with the url scheme e.g.: ws://1.2.3.4 or wss://1.2.3.4",
		},
		{
			name:       "wrong-scheme",
			value:      "http://1.2.3.4",
			errMessage: "invalid url scheme specified for --suiWS, try ws:// or wss://: http://1.2.3.4",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testCore, logs := observer.New(zap.InfoLevel)
			testLogger := zap.New(testCore)

			suiWatcher := Watcher{
				suiWS: testCase.value,
			}
			err := suiWatcher.fixSuiWsURL(testLogger)
			if testCase.errMessage != "" {
				require.EqualError(t, err, testCase.errMessage)
			} else {
				require.NoError(t, err)
				// Only verify the value if no error was returned
				assert.Equal(t, testCase.expected, suiWatcher.suiWS)
			}

			if len(testCase.logMessage) != 0 {
				// If the testcase expects a log, then there should only be 1 log
				require.Equal(t, 1, logs.Len())

				// Ensure the log message is correct
				actualLogMessage := logs.All()[0].Message
				require.Equal(t, testCase.logMessage, actualLogMessage)
			} else {
				// If the testcase does not expect a log, none should be emitted
				require.Equal(t, 0, logs.Len())
			}
		})
	}
}
