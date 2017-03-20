package client

import (
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/BlockchainSource/go-opentimestamps/opentimestamps"
	"github.com/btcsuite/btcrpcclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const envvarRPCURL = "GOTS_TEST_BITCOIN_RPC"

func newTestBTCConn() (*btcrpcclient.Client, error) {
	val := os.Getenv(envvarRPCURL)
	if val == "" {
		return nil, fmt.Errorf("envvar %q unset", envvarRPCURL)
	}
	connData, err := url.Parse(val)
	if err != nil {
		return nil, fmt.Errorf(
			"could not parse %q=%q: %v", envvarRPCURL, val, err,
		)
	}

	host := connData.Host
	if connData.User == nil {
		return nil, fmt.Errorf("no Userinfo in parsed url")
	}
	username := connData.User.Username()
	password, ok := connData.User.Password()
	if !ok {
		return nil, fmt.Errorf("no password given in RPC URL")
	}

	connCfg := &btcrpcclient.ConnConfig{
		Host:         host,
		User:         username,
		Pass:         password,
		HTTPPostMode: true,
		DisableTLS:   true,
	}
	return btcrpcclient.New(connCfg, nil)
}

func TestVerifyHelloWorld(t *testing.T) {
	if os.Getenv(envvarRPCURL) == "" {
		t.Skipf("envvar %s unset, skipping", envvarRPCURL)
	}

	// Format RFC3339
	expectedTime := "2015-05-28T15:41:18Z"

	helloWorld, err := opentimestamps.NewDetachedTimestampFromPath(
		"../../examples/hello-world.txt.ots",
	)
	require.NoError(t, err)
	ts := helloWorld.Timestamp

	btcConn, err := newTestBTCConn()
	require.NoError(t, err)

	verifier := BitcoinAttestationVerifier{btcConn}

	// using BitcoinVerifications()
	results := verifier.BitcoinVerifications(ts)
	assert.Equal(t, 1, len(results))
	result0 := results[0]
	require.NoError(t, result0.Error)
	assert.Equal(
		t, expectedTime, result0.AttestationTime.Format(time.RFC3339),
	)

	// using Verify()
	verifiedTime, err := verifier.Verify(ts)
	require.NoError(t, err)
	require.NotNil(t, verifiedTime)
	assert.Equal(t, expectedTime, verifiedTime.Format(time.RFC3339))
}
