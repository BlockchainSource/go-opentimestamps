package opentimestamps

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

type dumpConfig struct {
	showMessage bool
}

var defaultDumpConfig dumpConfig = dumpConfig{
	showMessage: true,
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
	message      []byte
	attestations []attestation
	ops          []tsLink
}

func (t *Timestamp) DumpIndent(w io.Writer, indent int, cfg dumpConfig) {
	if cfg.showMessage {
		fmt.Fprintf(w, strings.Repeat(" ", indent))
		fmt.Fprintf(w, "message %x\n", t.message)
	}
	for _, att := range t.attestations {
		fmt.Fprint(w, strings.Repeat(" ", indent))
		fmt.Fprintln(w, att)
	}

	// if the timestamp is indeed tree-shaped, show it like that
	if len(t.ops) > 1 {
		indent += 1
	}

	for _, tsLink := range t.ops {
		fmt.Fprint(w, strings.Repeat(" ", indent))
		fmt.Fprintln(w, tsLink.opCode)
		fmt.Fprint(w, strings.Repeat(" ", indent))
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
