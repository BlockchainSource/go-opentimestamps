package main

import (
	"flag"
	"log"
	"os"

	"github.com/BlockchainSource/go-opentimestamps/opentimestamps"
)

const defaultCalendar = "https://alice.btc.calendar.opentimestamps.org"

func main() {
	flag.Parse()
	path := flag.Arg(0)

	cal, err := opentimestamps.NewRemoteCalendar(defaultCalendar)
	if err != nil {
		log.Fatalf("error creating remote calendar: %v", err)
	}

	outFile, err := os.Create(path + ".ots")
	if err != nil {
		log.Fatalf("error creating output file: %v", err)
	}

	dts, err := opentimestamps.CreateDetachedTimestampForFile(path, cal)
	if err != nil {
		log.Fatalf(
			"error creating detached timestamp for %s: %v",
			path, err,
		)
	}
	if err := dts.WriteToStream(outFile); err != nil {
		log.Fatalf("error writing detached timestamp: %v", err)
	}
}
