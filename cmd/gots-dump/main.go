package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/BlockchainSource/go-opentimestamps/opentimestamps"
)

func main() {
	flag.Parse()
	path := flag.Arg(0)
	ts, err := opentimestamps.NewDetachedTimestampFromPath(path)
	if err != nil {
		log.Fatalf(
			"error reading detached timestamp %s: %v",
			path, err,
		)
	}

	fmt.Println(ts.Dump())
}
