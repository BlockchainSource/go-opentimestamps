package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/BlockchainSource/go-opentimestamps/opentimestamps"
)

func main() {
	flag.Parse()
	path := flag.Arg(0)
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}

	ts, err := opentimestamps.NewDetachedTimestampFile(f)
	if err != nil {
		log.Fatalf(
			"error decoding detached timestamp %s: %v",
			path, err,
		)
	}

	fmt.Println(ts.Dump())
}
