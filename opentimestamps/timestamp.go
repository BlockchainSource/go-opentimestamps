package opentimestamps

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// A timestampLink with the opCode being the link edge. The reference
// implementation uses a map, but the implementation is a bit complex. A list
// should work as well.
type tsLink struct {
	opCode    opCode
	timestamp *Timestamp
}

// A Timestamp can contain many attestations and operations.
type Timestamp struct {
	message      []byte
	attestations []attestation
	ops          []tsLink
}

func (t *Timestamp) DumpIndent(w io.Writer, indent int) {
	for _, att := range t.attestations {
		fmt.Fprint(w, strings.Repeat(" ", indent))
		fmt.Fprintln(w, att)
	}

	// if the timestamp is indeed tree-shaped, show it like that
	nextIndent := indent
	if len(t.ops) > 1 {
		nextIndent += 1
	}

	for _, tsLink := range t.ops {
		fmt.Fprint(w, strings.Repeat(" ", indent))
		fmt.Fprintln(w, tsLink.opCode)
		fmt.Fprint(w, strings.Repeat(" ", indent))
		tsLink.timestamp.DumpIndent(w, nextIndent)
	}
}

func (t *Timestamp) Dump() string {
	b := &bytes.Buffer{}
	t.DumpIndent(b, 0)
	return b.String()
}

func parseTagOrAttestation(
	ts *Timestamp,
	ctx *deserializationContext,
	tag byte,
	message []byte,
	limit int,
) error {
	if tag == 0x00 {
		a, err := ParseAttestation(ctx)
		if err != nil {
			return err
		}
		ts.attestations = append(ts.attestations, a)
	} else {
		op, err := parseOp(ctx, tag)
		if err != nil {
			return err
		}
		newMessage, err := op.apply(message)
		if err != nil {
			return err
		}
		nextTs := &Timestamp{message: newMessage}
		err = parse(nextTs, ctx, newMessage, limit-1)
		if err != nil {
			return err
		}
		ts.ops = append(ts.ops, tsLink{op, nextTs})

	}
	return nil
}

func parse(
	ts *Timestamp, ctx *deserializationContext, message []byte, limit int,
) error {
	if limit == 0 {
		return fmt.Errorf("recursion limit")
	}
	var tag byte
	var err error
	for {
		tag, err = ctx.readByte()
		if err != nil {
			return err
		}
		if tag == 0xff {
			tag, err = ctx.readByte()
			if err != nil {
				return err
			}
			err := parseTagOrAttestation(ts, ctx, tag, message, limit)
			if err != nil {
				return err
			}
		} else {
			break
		}
	}
	return parseTagOrAttestation(ts, ctx, tag, message, limit)
}

func newTimestampFromContext(
	ctx *deserializationContext, message []byte,
) (*Timestamp, error) {
	recursionLimit := 1000
	ts := &Timestamp{message: message}
	err := parse(ts, ctx, message, recursionLimit)
	if err != nil {
		return nil, err
	}
	return ts, nil
}

func NewTimestampFromReader(r io.Reader, message []byte) (*Timestamp, error) {
	return newTimestampFromContext(newDeserializationContext(r), message)
}
