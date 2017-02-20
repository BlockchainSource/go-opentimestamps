package opentimestamps

import "fmt"

const maxResultLength = 4096

type unaryStackOp func(message []byte) ([]byte, error)
type binaryStackOp func(message, argument []byte) ([]byte, error)

type opCode interface {
	match(byte) bool
	decode(*deserializationContext) (opCode, error)
	encode() error
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

func (u *unaryOp) encode() error {
	panic("not implemented")
}

func (u *unaryOp) apply(message []byte) ([]byte, error) {
	return u.stackOp(message)
}

func noop([]byte) ([]byte, error) { return nil, nil }

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
	ret := *b
	ret.argument = arg
	return &ret, nil
}

func (b *binaryOp) encode() error {
	panic("not implemented")
}

func (b *binaryOp) apply(message []byte) ([]byte, error) {
	return b.stackOp(message, b.argument)
}

func (b *binaryOp) String() string {
	return fmt.Sprintf("%s %x", b.name, b.argument)
}

func opAppend(msg, arg []byte) ([]byte, error)  { return nil, nil }
func opPrepend(msg, arg []byte) ([]byte, error) { return nil, nil }

var opCodes []opCode = []opCode{
	newBinaryOp(0xf0, "APPEND", opAppend),
	newBinaryOp(0xf1, "PREPEND", opPrepend),
	newUnaryOp(0xf2, "REVERSE", noop),
	newUnaryOp(0xf3, "HEXLIFY", noop),
	newCryptOp(0x02, "SHA1", noop, 20),
	newCryptOp(0x03, "RIPEMD160", noop, 20),
	newCryptOp(0x08, "SHA256", noop, 32),
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
