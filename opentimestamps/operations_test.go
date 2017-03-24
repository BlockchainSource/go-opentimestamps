package opentimestamps

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMsgAppend(t *testing.T) {
	msg := []byte("123")
	res, err := msgAppend(msg, []byte("456"))
	assert.NoError(t, err)
	assert.Equal(t, "123456", string(res))
	// make sure changes to input msg don't affect output
	msg[0] = byte('0')
	assert.Equal(t, "123456", string(res))
}

func TestMsgPrepend(t *testing.T) {
	msg := []byte("123")
	res, err := msgPrepend(msg, []byte("abc"))
	assert.NoError(t, err)
	assert.Equal(t, "abc123", string(res))
	// make sure changes to input msg don't affect output
	msg[0] = byte('0')
	assert.Equal(t, "abc123", string(res))
}

func TestMsgReverse(t *testing.T) {
	_, err := msgReverse([]byte{})
	assert.Error(t, err)
	res, err := msgReverse([]byte{1, 2, 3})
	assert.NoError(t, err)
	assert.Equal(t, []byte{3, 2, 1}, res)
}

func TestMsgHexlify(t *testing.T) {
	_, err := msgHexlify([]byte{})
	assert.Error(t, err)
	res, err := msgHexlify([]byte{1, 2, 3, 0xff})
	assert.NoError(t, err)
	assert.Equal(t, []byte("010203ff"), res)
}

func TestMsgSHA1(t *testing.T) {
	out, err := msgSHA1([]byte{})
	assert.NoError(t, err)
	assert.Equal(t,
		"da39a3ee5e6b4b0d3255bfef95601890afd80709",
		hex.EncodeToString(out),
	)
}

func TestMsgSHA256(t *testing.T) {
	out, err := msgSHA256([]byte{})
	assert.NoError(t, err)
	assert.Equal(t,
		"e3b0c44298fc1c149afbf4c8996fb924"+
			"27ae41e4649b934ca495991b7852b855",
		hex.EncodeToString(out),
	)
}

func TestRIPEMD160(t *testing.T) {
	out, err := msgRIPEMD160([]byte{})
	assert.Equal(t,
		"9c1185a5c5e9fc54612808977ee8f548b2258d31",
		hex.EncodeToString(out),
	)

	out, err = msgRIPEMD160(out)
	assert.NoError(t, err)
	assert.Equal(t,
		"38bbc57e4cbe8b6a1d2c999ef62503e0a6e58109",
		hex.EncodeToString(out),
	)
}
