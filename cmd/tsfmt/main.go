// Command tsfmt validates a time sheet file and prints it back out
// grouped by day with daily and grand totals.
package main

import (
	"fmt"
	"os"

	timesheet "github.com/johny380/timesheet-fmt"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: tsfmt <file>")
		os.Exit(2)
	}

	f, err := os.Open(os.Args[1])
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

	fmt.Print(timesheet.Format(sheet))
}
