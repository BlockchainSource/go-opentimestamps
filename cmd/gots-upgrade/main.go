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
	dts, err := opentimestamps.NewDetachedTimestampFromPath(path)
	if err != nil {
		log.Fatalf(
			"error reading detached timestamp %s: %v",
			path, err,
		)
	}

	var upgraded *opentimestamps.Timestamp

	for n, pts := range opentimestamps.PendingTimestamps(dts.Timestamp) {
		fmt.Printf(
			"#%2d: upgrade %v\n     %x\n    ",
			n, pts.PendingAttestation, pts.Timestamp.Message,
		)
		u, err := pts.Upgrade()
		if err != nil {
			fmt.Printf(" error %v", err)
		} else {
			fmt.Printf(" success")
		}
		fmt.Print("\n")

		// FIXME merge timestamp instead of replacing it
		upgraded = u
		break
	}

	if upgraded == nil {
		log.Fatal("no pending timestamps found")
	}

	dts.Timestamp = upgraded
	f, err := os.Create(path)
	if err != nil {
		log.Fatalf("error opening output file: %v", err)
	}
	defer f.Close()
	if err := dts.WriteToStream(f); err != nil {
		log.Fatalf("error writing detached timestamp: %v", err)
	}
	log.Print("timestamp updated successfully")
}
