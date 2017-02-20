package opentimestamps

import (
	"crypto/sha256"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const calendarServerEnvvar = "GOTS_TEST_CALENDAR_SERVER"
const bitcoinRegtestEnvvar = "GOTS_TEST_BITCOIN_REGTEST_SERVER"

func newTestCalendar(url string) *RemoteCalendar {
	logrus.SetLevel(logrus.DebugLevel)
	cal, err := NewRemoteCalendar(url)
	if err != nil {
		panic("could not create test calendar")
	}
	cal.log.Level = logrus.DebugLevel
	return cal
}

func newTestDigest(in string) []byte {
	hash := sha256.Sum256([]byte(in))
	return hash[:]
}

func TestRemoteCalendarExample(t *testing.T) {
	dts, err := NewDetachedTimestampFromPath(
		"../examples/two-calendars.txt.ots",
	)
	require.NoError(t, err)

	pts := PendingTimestamps(dts.Timestamp)
	assert.Equal(t, 2, len(pts))
	for _, pt := range pts {
		ts, err := pt.Upgrade()
		assert.NoError(t, err)
		fmt.Print(ts.Dump())
	}
}

func TestRemoteCalendarRoundTrip(t *testing.T) {
	calendarServer := os.Getenv(calendarServerEnvvar)
	if calendarServer == "" {
		t.Skipf("%q not set, skipping test", calendarServerEnvvar)
	}
	cal := newTestCalendar(calendarServer)
	ts, err := cal.Submit(newTestDigest("Hello, World!"))
	require.NoError(t, err)
	require.NotNil(t, ts)

	// TODO call btcrpcclient generateblock 100

	// FIXME possible opentimestamps-server bug?
	// wait until attestation has been aggregated
	time.Sleep(2 * time.Second)

	for _, pts := range PendingTimestamps(ts) {
		ts, err := pts.Upgrade()
		assert.NoError(t, err)
		_ = ts
	}
}
