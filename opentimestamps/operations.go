package opentimestamps

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/ripemd160"
)

const maxResultLength = 4096

type unaryStackOp func(message []byte) ([]byte, error)
type binaryStackOp func(message, argument []byte) ([]byte, error)

// opAppend returns the concatenation of msg and arg
func opAppend(msg, arg []byte) (res []byte, err error) {
	res = append(res, msg...)
	res = append(res, arg...)
	return
}

// opPrepend returns the concatenation of arg and msg
func opPrepend(msg, arg []byte) (res []byte, err error) {
	res = append(res, arg...)
	res = append(res, msg...)
	return
}

// opReverse returns the reversed msg. Deprecated.
func opReverse(msg []byte) ([]byte, error) {
	if len(msg) == 0 {
		return nil, fmt.Errorf("empty input invalid for opReverse")
	}
	res := make([]byte, len(msg))
	for i, b := range msg {
		res[len(res)-i-1] = b
	}
	return res, nil
}

func opHexlify(msg []byte) ([]byte, error) {
	if len(msg) == 0 {
		return nil, fmt.Errorf("empty input invalid for opHexlify")
	}
	return []byte(hex.EncodeToString(msg)), nil
}

func opSHA1(msg []byte) ([]byte, error) {
	res := sha1.Sum(msg)
	return res[:], nil
}

func opRIPEMD160(msg []byte) ([]byte, error) {
	h := ripemd160.New()
	_, err := h.Write(msg)
	if err != nil {
		return nil, err
	}
	return h.Sum([]byte{}), nil
}

func opSHA256(msg []byte) ([]byte, error) {
	res := sha256.Sum256(msg)
	return res[:], nil
}

type opCode interface {
	match(byte) bool
	decode(*deserializationContext) (opCode, error)
	encode(*serializationContext) error
	apply(message []byte) ([]byte, error)
}

type op struct {
	tag  byte
	name string
}

func (o op) match(tag byte) bool {
	return o.tag == tag
}

type unaryOp struct {
	op
	stackOp unaryStackOp
}

func newUnaryOp(tag byte, name string, stackOp unaryStackOp) *unaryOp {
	return &unaryOp{op{tag: tag, name: name}, stackOp}
}

func (u *unaryOp) String() string {
	return u.name
}

func (u *unaryOp) decode(ctx *deserializationContext) (opCode, error) {
	ret := *u
	return &ret, nil
}

func (u *unaryOp) encode(ctx *serializationContext) error {
	return ctx.writeByte(u.tag)
}

func (u *unaryOp) apply(message []byte) ([]byte, error) {
	return u.stackOp(message)
}

// Crypto operations
// These are hash ops that define a digest length
type cryptOp struct {
	unaryOp
	digestLength int
}

func newCryptOp(
	tag byte, name string, stackOp unaryStackOp, digestLength int,
) *cryptOp {
	return &cryptOp{
		unaryOp:      *newUnaryOp(tag, name, stackOp),
		digestLength: digestLength,
	}
}

func (c *cryptOp) decode(ctx *deserializationContext) (opCode, error) {
	u, err := c.unaryOp.decode(ctx)
	if err != nil {
		return nil, err
	}
	return &cryptOp{*u.(*unaryOp), c.digestLength}, nil
}

// Binary operations
// We decode an extra varbyte argument and use it in apply()

type binaryOp struct {
	op
	stackOp  binaryStackOp
	argument []byte
}

func newBinaryOp(tag byte, name string, stackOp binaryStackOp) *binaryOp {
	return &binaryOp{
		op:       op{tag: tag, name: name},
		stackOp:  stackOp,
		argument: nil,
	}
}

func (b *binaryOp) decode(ctx *deserializationContext) (opCode, error) {
	arg, err := ctx.readVarBytes(0, maxResultLength)
	if err != nil {
		return nil, err
	}
	if len(arg) == 0 {
		return nil, fmt.Errorf("empty argument invalid for binaryOp")
	}
	ret := *b
	ret.argument = arg
	return &ret, nil
}

func (b *binaryOp) encode(ctx *serializationContext) error {
	if err := ctx.writeByte(b.tag); err != nil {
		return err
	}
	return ctx.writeVarBytes(b.argument)
}

func (b *binaryOp) apply(message []byte) ([]byte, error) {
	return b.stackOp(message, b.argument)
}

func (b *binaryOp) String() string {
	return fmt.Sprintf("%s %x", b.name, b.argument)
}

var opCodes []opCode = []opCode{
	newBinaryOp(0xf0, "APPEND", opAppend),
	newBinaryOp(0xf1, "PREPEND", opPrepend),
	newUnaryOp(0xf2, "REVERSE", opReverse),
	newUnaryOp(0xf3, "HEXLIFY", opHexlify),
	newCryptOp(0x02, "SHA1", opSHA1, 20),
	newCryptOp(0x03, "RIPEMD160", opRIPEMD160, 20),
	newCryptOp(0x08, "SHA256", opSHA256, 32),
}

func parseOp(ctx *deserializationContext, tag byte) (opCode, error) {
	for _, op := range opCodes {
		if op.match(tag) {
			return op.decode(ctx)
		}
	}
	return nil, fmt.Errorf("could not decode tag %02x", tag)
}

func parseCryptOp(ctx *deserializationContext) (*cryptOp, error) {
	tag, err := ctx.readByte()
	if err != nil {
		return nil, err
	}
	op, err := parseOp(ctx, tag)
	if err != nil {
		return nil, err
	}
	if cryptOp, ok := op.(*cryptOp); ok {
		return cryptOp, nil
	} else {
		return nil, fmt.Errorf("expected cryptOp, got %#v", op)
	}
}
