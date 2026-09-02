package connectors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testRPCURLConnector struct {
	*mockConnectorForPoller
	rpcURL string
}

func (t *testRPCURLConnector) RPCURL() string {
	return t.rpcURL
}

func TestSafeConnectorErrorForLogging(t *testing.T) {
	rawURL := "https://user:pass@example.com/path?api_key=secret"
	err := errors.New("request failed for " + rawURL)

	assert.Equal(t, "", safeConnectorErrorForLogging(nil, &mockConnectorForPoller{}))
	assert.Equal(t, err.Error(), safeConnectorErrorForLogging(err, &mockConnectorForPoller{}))

	connector := &testRPCURLConnector{
		mockConnectorForPoller: &mockConnectorForPoller{},
		rpcURL:                 rawURL,
	}
	assert.Equal(t, "request failed for example.com", safeConnectorErrorForLogging(err, connector))
}
