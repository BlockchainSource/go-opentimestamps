package opentimestamps

import (
	"bytes"
	"fmt"
)

const (
	attestationTagSize             = 8
	attestationMaxPayloadSize      = 8192
	pendingAttestationMaxUriLength = 1000
)

var (
	bitcoinAttestationTag = mustDecodeHex("0588960d73d71901")
	pendingAttestationTag = mustDecodeHex("83dfe30d2ef90c8e")
)

type Attestation interface {
	match(tag []byte) bool
	decode(*deserializationContext) (Attestation, error)
}

type baseAttestation struct {
	tag []byte
}

func (b *baseAttestation) match(tag []byte) bool {
	return bytes.Equal(b.tag, tag)
}

type pendingAttestation struct {
	baseAttestation
	uri string
}

func newPendingAttestation() *pendingAttestation {
	return &pendingAttestation{
		baseAttestation: baseAttestation{tag: pendingAttestationTag},
	}
}

func (p *pendingAttestation) match(tag []byte) bool {
	return p.baseAttestation.match(tag)
}

func (p *pendingAttestation) decode(
	ctx *deserializationContext,
) (Attestation, error) {
	uri, err := ctx.readVarBytes(0, pendingAttestationMaxUriLength)
	if err != nil {
		return nil, err
	}
	// TODO utf8 checks
	ret := *p
	ret.uri = string(uri)
	return &ret, nil
}

func (p *pendingAttestation) String() string {
	return fmt.Sprintf("VERIFY PendingAttestation(url=%s)", p.uri)
}

type BitcoinAttestation struct {
	baseAttestation
	Height uint64
}

func newBitcoinAttestation() *BitcoinAttestation {
	return &BitcoinAttestation{
		baseAttestation: baseAttestation{bitcoinAttestationTag},
	}
}

func (b *BitcoinAttestation) String() string {
	return fmt.Sprintf("VERIFY BitcoinAttestation(height=%d)", b.Height)
}

func (b *BitcoinAttestation) match(tag []byte) bool {
	return b.baseAttestation.match(tag)
}

func (b *BitcoinAttestation) decode(
	ctx *deserializationContext,
) (Attestation, error) {
	height, err := ctx.readVarUint()
	if err != nil {
		return nil, err
	}
	ret := *b
	ret.Height = height
	return &ret, nil
}

// This is a catch-all for when we don't know how to parse it
type unknownAttestation struct {
	tag   []byte
	bytes []byte
}

func (u unknownAttestation) match(tag []byte) bool {
	panic("not implemented")
}

func (u unknownAttestation) decode(*deserializationContext) (Attestation, error) {
	panic("not implemented")
}

func (u unknownAttestation) String() string {
	return fmt.Sprintf("UnknownAttestation(bytes=%q)", u.bytes)
}

var attestations []Attestation = []Attestation{
	newPendingAttestation(),
	newBitcoinAttestation(),
}

func ParseAttestation(ctx *deserializationContext) (Attestation, error) {
	tag, err := ctx.readBytes(attestationTagSize)
	if err != nil {
		return nil, err
	}

	attBytes, err := ctx.readVarBytes(
		0, attestationMaxPayloadSize,
	)
	if err != nil {
		return nil, err
	}
	attCtx := newDeserializationContext(
		bytes.NewBuffer(attBytes),
	)

	for _, a := range attestations {
		if a.match(tag) {
			att, err := a.decode(attCtx)
			if err != nil {
				return nil, err
			}
			if !attCtx.assertEOF() {
				return nil, fmt.Errorf("expected EOF in attCtx")
			}
			return att, nil
		}
	}
	return unknownAttestation{tag, attBytes}, nil
}
