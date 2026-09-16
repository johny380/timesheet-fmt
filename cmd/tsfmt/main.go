// Command tsfmt validates a time sheet file and prints it back out
// grouped by day with daily and grand totals.
package main

import (
	"flag"
	"fmt"
	"os"

	timesheet "github.com/johny380/timesheet-fmt"
)

func main() {
	strict := flag.Bool("strict", false, "warn on gaps between entries within a day")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: tsfmt [-strict] <file>")
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()

	sheet, err := timesheet.Parse(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *strict {
		for _, g := range timesheet.FindGaps(sheet) {
			fmt.Fprintf(os.Stderr, "warning: %s\n", g)
		}
	}

	fmt.Print(timesheet.Format(sheet))
}
