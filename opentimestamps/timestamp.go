package opentimestamps

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

type dumpConfig struct {
	showMessage bool
	showFlat    bool
}

var defaultDumpConfig dumpConfig = dumpConfig{
	showMessage: true,
	showFlat:    false,
}

// A timestampLink with the opCode being the link edge. The reference
// implementation uses a map, but the implementation is a bit complex. A list
// should work as well.
type tsLink struct {
	opCode    opCode
	timestamp *Timestamp
}

// A Timestamp can contain many attestations and operations.
type Timestamp struct {
	Message      []byte
	Attestations []Attestation
	ops          []tsLink
}

// Walk calls the passed function f for this timestamp and all
// downstream timestamps that are chained via operations.
func (t *Timestamp) Walk(f func(t *Timestamp)) {
	f(t)
	for _, l := range t.ops {
		l.timestamp.Walk(f)
	}
}

func (t *Timestamp) encode(ctx *serializationContext) error {
	n := len(t.Attestations) + len(t.ops)
	if n == 0 {
		return fmt.Errorf("cannot encode empty timestamp")
	}
	prefixAtt := []byte{0x00}
	prefixOp := []byte{}
	nextNode := func(prefix []byte) error {
		n -= 1
		if n > 0 {
			return ctx.writeByte(0xff)
		}
		if len(prefix) > 0 {
			return ctx.writeBytes(prefix)
		}
		return nil
	}
	// FIXME attestations should be sorted
	for _, att := range t.Attestations {
		if err := nextNode(prefixAtt); err != nil {
			return err
		}
		if err := encodeAttestation(ctx, att); err != nil {
			return err
		}
	}
	// FIXME ops should be sorted
	for _, op := range t.ops {
		if err := nextNode(prefixOp); err != nil {
			return err
		}
		if err := op.opCode.encode(ctx); err != nil {
			return err
		}
		if err := op.timestamp.encode(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (t *Timestamp) DumpIndent(w io.Writer, indent int, cfg dumpConfig) {
	if cfg.showMessage {
		fmt.Fprintf(w, strings.Repeat(" ", indent))
		fmt.Fprintf(w, "message %x\n", t.Message)
	}
	for _, att := range t.Attestations {
		fmt.Fprint(w, strings.Repeat(" ", indent))
		fmt.Fprintln(w, att)
	}

	for _, tsLink := range t.ops {
		fmt.Fprint(w, strings.Repeat(" ", indent))
		fmt.Fprintln(w, tsLink.opCode)
		// fmt.Fprint(w, strings.Repeat(" ", indent))
		// if the timestamp is indeed tree-shaped, show it like that
		if !cfg.showFlat || len(t.ops) > 1 {
			indent += 1
		}
		tsLink.timestamp.DumpIndent(w, indent, cfg)
	}
}

func (t *Timestamp) DumpWithConfig(cfg dumpConfig) string {
	b := &bytes.Buffer{}
	t.DumpIndent(b, 0, cfg)
	return b.String()
}

func (t *Timestamp) Dump() string {
	return t.DumpWithConfig(defaultDumpConfig)
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
		ts.Attestations = append(ts.Attestations, a)
	} else {
		op, err := parseOp(ctx, tag)
		if err != nil {
			return err
		}
		newMessage, err := op.apply(message)
		if err != nil {
			return err
		}
		nextTs := &Timestamp{Message: newMessage}
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
	ts := &Timestamp{Message: message}
	err := parse(ts, ctx, message, recursionLimit)
	if err != nil {
		return nil, err
	}
	return ts, nil
}

func NewTimestampFromReader(r io.Reader, message []byte) (*Timestamp, error) {
	return newTimestampFromContext(newDeserializationContext(r), message)
}
