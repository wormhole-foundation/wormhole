package wormconn

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func TestNewTransportCredentials(t *testing.T) {
	target, creds := newTransportCredentials("wormchain:9090")
	assert.Equal(t, "wormchain:9090", target)
	assert.IsType(t, insecure.NewCredentials(), creds)

	target, creds = newTransportCredentials("https://wormchain:443")
	assert.Equal(t, "wormchain:443", target)
	assert.IsType(t, credentials.NewTLS(nil), creds)
}
