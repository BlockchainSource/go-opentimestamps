package opentimestamps

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func examplePaths() []string {
	matches, err := filepath.Glob("../examples/*ots")
	if err != nil {
		panic(err)
	}
	return matches
}

func containsUnknownAttestation(ts *Timestamp) (res bool) {
	ts.Walk(func(subTs *Timestamp) {
		for _, att := range subTs.Attestations {
			if _, ok := att.(unknownAttestation); ok {
				res = true
			}
		}
	})
	return
}

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

func TestDecodeEncodeAll(t *testing.T) {
	for _, path := range examplePaths() {
		t.Log(path)
		dts, err := NewDetachedTimestampFromPath(path)
		assert.NoError(t, err, path)

		if containsUnknownAttestation(dts.Timestamp) {
			t.Logf("skipping encode cycle: unknownAttestation")
			continue
		}

		buf := &bytes.Buffer{}
		err = dts.Timestamp.encode(&serializationContext{buf})
		if !assert.NoError(t, err, path) {
			continue
		}

		buf = bytes.NewBuffer(buf.Bytes())
		ts1, err := NewTimestampFromReader(buf, dts.Timestamp.Message)
		if !assert.NoError(t, err, path) {
			continue
		}

		dts1, err := NewDetachedTimestamp(
			dts.HashOp, dts.FileHash, ts1,
		)
		if !assert.NoError(t, err) {
			continue
		}

		dts1Target := &bytes.Buffer{}
		err = dts1.WriteToStream(dts1Target)
		if !assert.NoError(t, err) {
			continue
		}

		orgBytes, err := ioutil.ReadFile(path)
		if !assert.NoError(t, err) {
			continue
		}

		assert.Equal(t, orgBytes, dts1Target.Bytes())
		t.Log("encode cycle success")
	}
}
