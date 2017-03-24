package opentimestamps

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/ripemd160"
)

const maxResultLength = 4096

type unaryMsgOp func(message []byte) ([]byte, error)
type binaryMsgOp func(message, argument []byte) ([]byte, error)

// msgAppend returns the concatenation of msg and arg
func msgAppend(msg, arg []byte) (res []byte, err error) {
	res = append(res, msg...)
	res = append(res, arg...)
	return
}

// msgPrepend returns the concatenation of arg and msg
func msgPrepend(msg, arg []byte) (res []byte, err error) {
	res = append(res, arg...)
	res = append(res, msg...)
	return
}

// msgReverse returns the reversed msg. Deprecated.
func msgReverse(msg []byte) ([]byte, error) {
	if len(msg) == 0 {
		return nil, fmt.Errorf("empty input invalid for msgReverse")
	}
	res := make([]byte, len(msg))
	for i, b := range msg {
		res[len(res)-i-1] = b
	}
	return res, nil
}

func msgHexlify(msg []byte) ([]byte, error) {
	if len(msg) == 0 {
		return nil, fmt.Errorf("empty input invalid for msgHexlify")
	}
	return []byte(hex.EncodeToString(msg)), nil
}

func msgSHA1(msg []byte) ([]byte, error) {
	res := sha1.Sum(msg)
	return res[:], nil
}

func msgRIPEMD160(msg []byte) ([]byte, error) {
	h := ripemd160.New()
	_, err := h.Write(msg)
	if err != nil {
		return nil, err
	}
	return h.Sum([]byte{}), nil
}

func msgSHA256(msg []byte) ([]byte, error) {
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
	msgOp unaryMsgOp
}

func newUnaryOp(tag byte, name string, msgOp unaryMsgOp) *unaryOp {
	return &unaryOp{op{tag: tag, name: name}, msgOp}
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
	return u.msgOp(message)
}

// Crypto operations
// These are hash ops that define a digest length
type cryptOp struct {
	unaryOp
	digestLength int
}

func newCryptOp(
	tag byte, name string, msgOp unaryMsgOp, digestLength int,
) *cryptOp {
	return &cryptOp{
		unaryOp:      *newUnaryOp(tag, name, msgOp),
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
	msgOp    binaryMsgOp
	argument []byte
}

func newBinaryOp(tag byte, name string, msgOp binaryMsgOp) *binaryOp {
	return &binaryOp{
		op:       op{tag: tag, name: name},
		msgOp:    msgOp,
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
	return b.msgOp(message, b.argument)
}

func (b *binaryOp) String() string {
	return fmt.Sprintf("%s %x", b.name, b.argument)
}

var (
	opAppend    = newBinaryOp(0xf0, "APPEND", msgAppend)
	opPrepend   = newBinaryOp(0xf1, "PREPEND", msgPrepend)
	opReverse   = newUnaryOp(0xf2, "REVERSE", msgReverse)
	opHexlify   = newUnaryOp(0xf3, "HEXLIFY", msgHexlify)
	opSHA1      = newCryptOp(0x02, "SHA1", msgSHA1, 20)
	opRIPEMD160 = newCryptOp(0x03, "RIPEMD160", msgRIPEMD160, 20)
	opSHA256    = newCryptOp(0x08, "SHA256", msgSHA256, 32)
)

var opCodes []opCode = []opCode{
	opAppend, opPrepend, opReverse, opHexlify, opSHA1, opRIPEMD160,
	opSHA256,
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
