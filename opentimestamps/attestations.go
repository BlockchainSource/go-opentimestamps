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
	tag() []byte
	decode(*deserializationContext) (Attestation, error)
	encode(*serializationContext) error
}

type baseAttestation struct {
	fixedTag []byte
}

func (b *baseAttestation) tag() []byte {
	return b.fixedTag
}

type pendingAttestation struct {
	baseAttestation
	uri string
}

func newPendingAttestation() *pendingAttestation {
	return &pendingAttestation{
		baseAttestation: baseAttestation{
			fixedTag: pendingAttestationTag,
		},
	}
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

func (p *pendingAttestation) encode(ctx *serializationContext) error {
	return ctx.writeVarBytes([]byte(p.uri))
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

func (b *BitcoinAttestation) encode(ctx *serializationContext) error {
	return ctx.writeVarUint(uint64(b.Height))
}

const hashMerkleRootSize = 32

//
func (b *BitcoinAttestation) VerifyAgainstBlockHash(
	digest, blockHash []byte,
) error {
	if len(digest) != hashMerkleRootSize {
		return fmt.Errorf("invalid digest size %d", len(digest))
	}
	if !bytes.Equal(digest, blockHash) {
		return fmt.Errorf(
			"hash mismatch digest=%x blockHash=%x",
			digest, blockHash,
		)
	}
	return nil
}

// This is a catch-all for when we don't know how to parse it
type unknownAttestation struct {
	tagBytes []byte
	bytes    []byte
}

func (u unknownAttestation) tag() []byte {
	return u.tagBytes
}

func (unknownAttestation) decode(*deserializationContext) (Attestation, error) {
	panic("not implemented")
}

func (unknownAttestation) encode(*serializationContext) error {
	panic("not implemented")
}

func (u unknownAttestation) String() string {
	return fmt.Sprintf("UnknownAttestation(bytes=%q)", u.bytes)
}

var attestations []Attestation = []Attestation{
	newPendingAttestation(),
	newBitcoinAttestation(),
}

func encodeAttestation(ctx *serializationContext, att Attestation) error {
	if err := ctx.writeBytes(att.tag()); err != nil {
		return err
	}
	buf := &bytes.Buffer{}
	if err := att.encode(&serializationContext{buf}); err != nil {
		return err
	}
	return ctx.writeVarBytes(buf.Bytes())
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
		if bytes.Equal(tag, a.tag()) {
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
