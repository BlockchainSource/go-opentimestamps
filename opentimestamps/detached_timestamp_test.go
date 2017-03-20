package opentimestamps

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeHelloWorld(t *testing.T) {
	dts, err := NewDetachedTimestampFromPath(
		"../examples/hello-world.txt.ots",
	)
	assert.NoError(t, err)

	attCount := 0
	checkAttestation := func(ts *Timestamp, att Attestation) {
		assert.Equal(t, 0, attCount)

		expectedAtt := newBitcoinAttestation()
		expectedAtt.Height = 358391
		assert.Equal(t, expectedAtt, att)

		// If ts.Message is correct, opcode parsing and execution should
		// have succeeded.
		assert.Equal(t,
			"007ee445d23ad061af4a36b809501fab1ac4f2d7e7a739817dd0cbb7ec661b8a",
			hex.EncodeToString(ts.Message),
		)

		attCount += 1
	}

	dts.Timestamp.Walk(func(ts *Timestamp) {
		for _, att := range ts.Attestations {
			// this should be called exactly once
			checkAttestation(ts, att)
		}
	})

	assert.Equal(t, 1, attCount)
}
