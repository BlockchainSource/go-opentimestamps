package opentimestamps

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeHelloWorld(t *testing.T) {
	_, err := NewDetachedTimestampFromPath(
		"../examples/hello-world.txt.ots",
	)
	assert.NoError(t, err)
}
