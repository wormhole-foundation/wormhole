package devnet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFirstGuardianNameFromBootstrapPeers(t *testing.T) {
	type test struct {
		label            string
		bootstrapPeers   string
		errText          string // empty string means success
		expectedHostName string
	}

	var tests = []test{
		{
			label:            "Success with one bootstrap peer",
			bootstrapPeers:   "/dns4/guardian-0.guardian/udp/8999/quic/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jw",
			errText:          "",
			expectedHostName: "guardian-0.guardian",
		},
		{
			label:            "Success with two bootstrap peer",
			bootstrapPeers:   "/dns4/guardian-0.guardian/udp/8999/quic/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jw,/dns4/guardian-1.guardian/udp/8999/quic/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jx",
			errText:          "",
			expectedHostName: "guardian-0.guardian",
		},
		{
			label:            "Success when using IP",
			bootstrapPeers:   "/dns4/10.121.2.4/udp/8999/quic/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jw",
			errText:          "",
			expectedHostName: "10.121.2.4",
		},
		{
			label:            "Empty bootstrap peers",
			bootstrapPeers:   "",
			errText:          "failed to parse devnet first bootstrap peer",
			expectedHostName: "",
		},
		{
			label:            "Empty first bootstrap peer",
			bootstrapPeers:   ",/dns4/guardian-0.guardian/udp/8999/quic/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jw",
			errText:          "failed to parse devnet first bootstrap peer",
			expectedHostName: "",
		},
		{
			label:            "No slashes",
			bootstrapPeers:   ":dns4:10.121.2.4:udp:8999:quic:p2p:12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jw",
			errText:          "failed to parse devnet first bootstrap peer",
			expectedHostName: "",
		},
	}

	for _, tst := range tests {
		t.Run(tst.label, func(t *testing.T) {
			hostName, err := GetFirstGuardianNameFromBootstrapPeers(tst.bootstrapPeers)
			if tst.errText == "" {
				require.NoError(t, err)
				assert.Equal(t, tst.expectedHostName, hostName)
			} else {
				require.ErrorContains(t, err, tst.errText)
			}
		})
	}

}
