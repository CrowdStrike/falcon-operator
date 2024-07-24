package falcon_registry

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHttpTransport(t *testing.T) {
	transport := newHttpTransport()
	assert.NotNil(t, transport.Proxy, "missing proxy configuration")
	require.NotNil(t, transport.TLSClientConfig, "missing TLS client config")
	assert.Equal(t, uint16(tls.VersionTLS12), transport.TLSClientConfig.MinVersion, "wrong minimum TLS version")
}
