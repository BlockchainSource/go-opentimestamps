package opentimestamps

import (
	"bytes"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newDeserializationContextFromBytes(in []byte) *deserializationContext {
	return newDeserializationContext(bytes.NewBuffer(in))
}

func TestReadWrite(t *testing.T) {
	magic := []byte("magic")
	buf := &bytes.Buffer{}
	s := newSerializationContext(buf)

	assert.NoError(t, s.writeBytes([]byte{0x00, 0x01}))
	assert.NoError(t, s.writeByte(0x02))
	assert.NoError(t, s.writeBool(true))
	assert.NoError(t, s.writeBool(false))
	assert.NoError(t, s.writeByte(0x03))
	assert.NoError(t, s.writeVarUint(1))
	assert.NoError(t, s.writeBytes([]byte{0x81, 0x00}))
	assert.NoError(t, s.writeBytes([]byte{0x81, 0x01}))
	assert.NoError(t, s.writeVarUint(0x100))
	assert.NoError(t, s.writeVarUint(uint64(math.MaxUint32)+1))
	assert.NoError(t, s.writeVarUint(math.MaxUint64))
	assert.NoError(t, s.writeBytes([]byte{
		// varunit excess MaxUint64
		0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0x01,
	}))
	assert.NoError(t, s.writeBytes(magic))
	assert.NoError(t, s.writeByte(0))
	assert.NoError(t, s.writeBytes(magic))

	data := buf.Bytes()

	expectedData := []byte{
		0x00, 0x01, // bytes [0x00, 0x01]
		0x02,       // byte 0x02
		0xff,       // bool true
		0x00,       // bool false
		0x03,       // bool error
		0x01,       // varuint 1
		0x81, 0x00, // varuint 1
		0x81, 0x01, // varuint 1 (alternative)
		0x80, 0x02, // varuint 0x100

		// varunit math.MaxUint32 + 1
		0x80, 0x80, 0x80, 0x80, 0x10,

		// varunit math.MaxUint64
		0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff,
		0xff, 0x01,

		// varunit excess math.MaxUint64
		0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0x01,

		// "magic"
		0x6d, 0x61, 0x67, 0x69, 0x63,
		// zero
		0x00,
		// "magic"
		0x6d, 0x61, 0x67, 0x69, 0x63,
	}

	assert.Equal(t, expectedData, data)

	d := newDeserializationContextFromBytes(data)

	{
		v, err := d.readBytes(2)
		assert.NoError(t, err)
		assert.Equal(t, []byte{0x00, 0x01}, v)
	}
	{
		v, err := d.readByte()
		assert.NoError(t, err)
		assert.Equal(t, byte(0x02), v)
	}
	{
		v, err := d.readBool()
		assert.NoError(t, err)
		assert.Equal(t, true, v)
	}
	{
		v, err := d.readBool()
		assert.NoError(t, err)
		assert.Equal(t, false, v)
	}
	{
		_, err := d.readBool()
		assert.Error(t, err)
	}
	{
		v, err := d.readVarUint()
		assert.NoError(t, err)
		assert.Equal(t, uint64(1), v)
	}
	{
		v, err := d.readVarUint()
		assert.NoError(t, err)
		assert.Equal(t, uint64(1), v)
	}
	{
		v, err := d.readVarUint()
		assert.NoError(t, err)
		assert.Equal(t, uint64(0x81), v)
	}
	{
		v, err := d.readVarUint()
		assert.NoError(t, err)
		assert.Equal(t, uint64(0x100), v)
	}
	{
		v, err := d.readVarUint()
		assert.NoError(t, err)
		assert.Equal(t, uint64(math.MaxUint32)+uint64(1), v)
	}
	{
		v, err := d.readVarUint()
		assert.NoError(t, err)
		assert.Equal(t, uint64(math.MaxUint64), uint64(v))
	}
	{
		_, err := d.readVarUint()
		assert.Error(t, err)
		// read leftover 0x02
		b, err := d.readByte()
		assert.NoError(t, err)
		assert.Equal(t, byte(0x01), b)

	}
	{
		assert.NoError(t, d.assertMagic(magic))
		// fails because of in-between 0x00
		assert.Error(t, d.assertMagic(magic))
	}
	{
		// read leftover byte
		_, err := d.readByte()
		assert.NoError(t, err)
		assert.True(t, d.assertEOF())
	}
}
