package clientcommon

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig_GetPeerTTL(t *testing.T) {
	tests := []struct {
		name     string
		peerTTL  time.Duration
		expected time.Duration
	}{
		{
			name:     "zero uses default",
			peerTTL:  0,
			expected: 2 * time.Minute,
		},
		{
			name:     "non-zero returned as-is",
			peerTTL:  5 * time.Minute,
			expected: 5 * time.Minute,
		},
		{
			name:     "negative uses default",
			peerTTL:  -1 * time.Second,
			expected: 2 * time.Minute,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{PeerTTL: tc.peerTTL}
			assert.Equal(t, tc.expected, cfg.GetPeerTTL())
		})
	}
}
