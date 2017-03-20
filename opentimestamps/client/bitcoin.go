package client

import (
	"fmt"
	"math"
	"time"

	"github.com/BlockchainSource/go-opentimestamps/opentimestamps"
	"github.com/btcsuite/btcrpcclient"
)

// A BitcoinAttestationVerifier uses a bitcoin RPC connection to verify bitcoin
// headers.
type BitcoinAttestationVerifier struct {
	btcrpcClient *btcrpcclient.Client
}

// VerifyAttestation checks a BitcoinAttestation using a given hash digest. It
// returns the time of the block if the verification succeeds, an error
// otherwise.
func (v *BitcoinAttestationVerifier) VerifyAttestation(
	digest []byte, a *opentimestamps.BitcoinAttestation,
) (*time.Time, error) {
	if a.Height > math.MaxInt64 {
		return nil, fmt.Errorf("illegal block height")
	}
	blockHash, err := v.btcrpcClient.GetBlockHash(int64(a.Height))
	if err != nil {
		return nil, err
	}
	h, err := v.btcrpcClient.GetBlockHeader(blockHash)
	if err != nil {
		return nil, err
	}

	merkleRootBytes := h.MerkleRoot[:]
	err = a.VerifyAgainstBlockHash(digest, merkleRootBytes)
	if err != nil {
		return nil, err
	}
	utc := h.Timestamp.UTC()

	return &utc, nil
}

// A BitcoinVerification is the result of verifying a BitcoinAttestation
type BitcoinVerification struct {
	Timestamp       *opentimestamps.Timestamp
	Attestation     *opentimestamps.BitcoinAttestation
	AttestationTime *time.Time
	Error           error
}

// BitcoinVerifications returns the all bitcoin attestation results for the
// timestamp.
func (v *BitcoinAttestationVerifier) BitcoinVerifications(
	t *opentimestamps.Timestamp,
) (res []BitcoinVerification) {
	t.Walk(func(ts *opentimestamps.Timestamp) {
		for _, att := range ts.Attestations {
			btcAtt, ok := att.(*opentimestamps.BitcoinAttestation)
			if !ok {
				continue
			}
			attTime, err := v.VerifyAttestation(ts.Message, btcAtt)
			res = append(res, BitcoinVerification{
				Timestamp:       ts,
				Attestation:     btcAtt,
				AttestationTime: attTime,
				Error:           err,
			})
		}
	})
	return res
}

// Verify returns the earliest bitcoin-attested time, or nil if none can be
// found or verified successfully.
func (v *BitcoinAttestationVerifier) Verify(
	t *opentimestamps.Timestamp,
) (ret *time.Time, err error) {
	res := v.BitcoinVerifications(t)
	for _, r := range res {
		if r.Error != nil {
			err = r.Error
			continue
		}
		if ret == nil || r.AttestationTime.Before(*ret) {
			ret = r.AttestationTime
		}
	}
	return
}
