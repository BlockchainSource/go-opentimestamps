package opentimestamps

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

var fileHeaderMagic = []byte(
	"\x00OpenTimestamps\x00\x00Proof\x00\xbf\x89\xe2\xe8\x84\xe8\x92\x94",
)

const minFileDigestLength = 20
const maxFileDigestLength = 32
const fileMajorVersion = 1

type DetachedTimestamp struct {
	HashOp    cryptOp
	Timestamp Timestamp
}

func (d *DetachedTimestamp) Dump() string {
	w := &bytes.Buffer{}
	fmt.Fprintf(
		w, "File %s hash: %x\n", d.HashOp.name, d.Timestamp.message,
	)
	fmt.Fprint(w, d.Timestamp.Dump())
	return w.String()
}

func NewDetachedTimestamp(r io.Reader) (*DetachedTimestamp, error) {
	ctx := newDeserializationContext(r)
	if err := ctx.assertMagic([]byte(fileHeaderMagic)); err != nil {
		return nil, err
	}
	major, err := ctx.readVarUint()
	if err != nil {
		return nil, err
	}
	if major != uint64(fileMajorVersion) {
		return nil, fmt.Errorf("unexpected major version %d", major)
	}
	fileHashOp, err := parseCryptOp(ctx)
	if err != nil {
		return nil, err
	}
	fileHash, err := ctx.readBytes(fileHashOp.digestLength)
	if err != nil {
		return nil, err
	}
	ts, err := newTimestampFromContext(ctx, fileHash)
	if err != nil {
		return nil, err
	}
	return &DetachedTimestamp{
		*fileHashOp, *ts,
	}, nil
}

func NewDetachedTimestampFromPath(p string) (*DetachedTimestamp, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	return NewDetachedTimestamp(f)
}
