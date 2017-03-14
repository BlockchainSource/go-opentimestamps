package opentimestamps

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testFile string

func (f testFile) DetachedTimestamp() (*DetachedTimestampFile, error) {
	r, err := os.Open(string(f) + ".ots")
	if err != nil {
		return nil, err
	}
	return NewDetachedTimestampFile(r)
}

var testFileHelloWorld = testFile("../examples/hello-world.txt")

func TestDecodeHelloWorld(t *testing.T) {
	_, err := testFileHelloWorld.DetachedTimestamp()
	assert.NoError(t, err)
}
