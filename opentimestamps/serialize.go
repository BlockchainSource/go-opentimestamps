package opentimestamps

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
)

// serializationContext helps encoding values in the ots format
type serializationContext struct {
	w io.Writer
}

// newSerializationContext returns a serializationContext for a writer
func newSerializationContext(w io.Writer) *serializationContext {
	return &serializationContext{w}
}

// writeBytes writes the raw bytes to the underlying writer
func (s serializationContext) writeBytes(b []byte) error {
	// number of bytes can be ignored
	// if it is equal len(b) then err is nil
	_, err := s.w.Write(b)
	if err != nil {
		return err
	}
	return nil
}

// writeByte writes a single byte
func (s serializationContext) writeByte(b byte) error {
	return s.writeBytes([]byte{b})
}

// writeBool encodes and writes a boolean value
func (s serializationContext) writeBool(b bool) error {
	if b {
		return s.writeByte(0xff)
	} else {
		return s.writeByte(0x00)
	}
}

// writeVarUint encodes and writes writes a variable-length integer
func (s serializationContext) writeVarUint(v uint64) error {
	if v == 0 {
		s.writeByte(0x00)
	}
	for v > 0 {
		b := byte(v & 0x7f)
		if v > uint64(0x7f) {
			b |= 0x80
		}
		if err := s.writeByte(b); err != nil {
			return err
		}
		if v <= 0x7f {
			break
		}
		v >>= 7
	}
	return nil
}

// writeVarBytes encodes and writes a variable-length array
func (s serializationContext) writeVarBytes(arr []byte) error {
	if err := s.writeVarUint(uint64(len(arr))); err != nil {
		return err
	}
	return s.writeBytes(arr)
}

// deserializationContext helps decoding values from the ots format
type deserializationContext struct {
	r io.Reader
}

// safety boundary for readBytes
// allocation limit for arrays
const maxReadSize = (1 << 12)

func (d deserializationContext) dump() string {
	arr, _ := d.r.(*bufio.Reader).Peek(512)
	return fmt.Sprintf("% x", arr)
}

// readBytes reads n bytes.
func (d deserializationContext) readBytes(n int) ([]byte, error) {
	if n > maxReadSize {
		return nil, fmt.Errorf("over maxReadSize: %d", maxReadSize)
	}
	b := make([]byte, n)
	m, err := d.r.Read(b)
	if err != nil {
		return b, err
	}
	if n != m {
		return b, fmt.Errorf("expected %d bytes, got %d", m, n)
	}
	return b[:], nil
}

// readByte reads a single byte.
func (d deserializationContext) readByte() (byte, error) {
	arr, err := d.readBytes(1)
	if err != nil {
		return 0, err
	}
	return arr[0], nil
}

// readBool reads a boolean.
func (d deserializationContext) readBool() (bool, error) {
	arr, err := d.readBytes(1)
	if err != nil {
		return false, err
	}
	switch v := arr[0]; v {
	case 0x00:
		return false, nil
	case 0xff:
		return true, nil
	default:
		return false, fmt.Errorf("unexpected value %x", v)
	}
}

// readVarUint reads a variable-length uint64.
func (d deserializationContext) readVarUint() (uint64, error) {
	// NOTE
	// the original python implementation has no uint64 limit, but I
	// don't think we'll ever need more that that.
	val := uint64(0)
	shift := uint(0)
	for {
		b, err := d.readByte()
		if err != nil {
			return 0, err
		}
		shifted := uint64(b&0x7f) << shift
		// ghetto overflow check
		if (shifted >> shift) != uint64(b&0x7f) {
			return 0, fmt.Errorf("uint64 overflow")
		}
		val |= shifted
		if b&0x80 == 0 {
			return val, nil
		}
		shift += 7
	}
}

// readVarBytes reads variable-length number of bytes.
func (d deserializationContext) readVarBytes(minLen, maxLen int) ([]byte, error) {
	v, err := d.readVarUint()
	if err != nil {
		return nil, err
	}
	if v > math.MaxInt32 {
		return nil, fmt.Errorf("int overflow")
	}
	vint := int(v)
	if maxLen < vint || vint < minLen {
		return nil, fmt.Errorf(
			"varbytes length %d outside range (%d, %d)",
			vint, minLen, maxLen,
		)
	}

	return d.readBytes(vint)
}

// assertMagic removes reads the expected bytes from the stream. Returns an
// error if the bytes are unexpected.
func (d deserializationContext) assertMagic(expected []byte) error {
	arr, err := d.readBytes(len(expected))
	if err != nil {
		return err
	}
	if !bytes.Equal(expected, arr) {
		return fmt.Errorf(
			"magic bytes mismatch, expected % x got % x",
			expected, arr,
		)
	}
	return nil
}

// newDeserializationContext returns a deserializationContext for a reader
func newDeserializationContext(r io.Reader) *deserializationContext {
	// TODO
	// bufio is used here to allow debugging via d.dump()
	// once this code here is robust enough we can just pass r
	return &deserializationContext{bufio.NewReader(r)}
}
