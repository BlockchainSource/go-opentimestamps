package opentimestamps

import "encoding/hex"

func mustDecodeHex(in string) []byte {
	out, err := hex.DecodeString(in)
	if err != nil {
		panic(err)
	}
	return out
}
