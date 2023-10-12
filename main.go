package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/danbrakeley/frog"
	"github.com/danbrakeley/sfv"
)

func main() {
	status := mainExit()
	if status != 0 {
		// From os/proc.go: "For portability, the status code should be in the range [0, 125]."
		if status < 0 || status > 125 {
			status = 125
		}
		os.Exit(status)
	}
}

func mainExit() int {
	flagJSON := flag.Bool("json", false, "output results as json")
	flag.Parse()

	var log frog.RootLogger
	if *flagJSON {
		log = frog.New(frog.Basic, frog.POTime(false), frog.POFieldIndent(20))
	} else {
		log = frog.New(frog.Auto, frog.POTime(false), frog.POFieldIndent(20))
	}
	defer log.Close()

	files := flag.Args()
	if len(files) != 1 {
		log.Error(fmt.Sprintf("usage: %s [-json] <file.sfv>", filepath.Base(os.Args[0])))
		return 1
	}

	line := frog.AddAnchor(log)
	line.Transient(fmt.Sprintf("Parsing %s...", files[0]))

	sf, err := sfv.CreateFromFile(files[0])
	if err != nil {
		frog.RemoveAnchor(line)
		log.Error("error parsing sfv file", frog.Err(err), frog.String("file", files[0]))
		return 1
	}

	fnProgress := func(filename string, read, total int64) {
		line.Transient(fmt.Sprintf("Checking %s %3d%%", filename, 100*read/total))
	}

	results := sf.Verify(fnProgress)
	frog.RemoveAnchor(line)

	if *flagJSON {
		b, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			log.Error("error generating json", frog.Err(err))
			return 1
		}
		fmt.Print(string(b))
		return 0
	}

	hasErrors := false
	for _, entry := range results.Files {
		if len(entry.Err) == 0 {
			log.Info("OK", frog.String("file", entry.Filename), frog.String("crc", entry.ActualCRC32))
		} else {
			log.Error("mismatch!",
				frog.String("file", entry.Filename),
				frog.String("expected_crc", entry.ExpectedCRC32),
				frog.String("actual_crc", entry.ActualCRC32),
				frog.String("error", entry.Err),
			)
			hasErrors = true
		}
	}

	if hasErrors {
		return 2
	}

	return 0
}
