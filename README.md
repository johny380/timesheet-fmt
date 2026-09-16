# timesheet-fmt

Time sheets kept as plain text are easy to write and hell to trust. A
missing minute turns 9:00-17:00 into 9:00-1700, a copy-pasted line
overlaps the one above it, a typo turns February into the 30th of
February. None of that shows up until someone tries to add the hours up.

This is a small validating parser and pretty printer for a plain-text
time sheet format, plus a CLI that runs both.

## Format

One entry per line:

```
DATE TIME-RANGE PROJECT: DESCRIPTION
```

```
# week of sept 1
2026-09-01 09:00-12:30 acme-corp: environment setup
2026-09-01 13:00-17:30 acme-corp: implement parser
2026-09-02 09:00-17:00 internal: planning
```

Blank lines and lines starting with `#` are ignored. Dates are
`YYYY-MM-DD`, times are 24-hour `HH:MM`. A line fails to parse if:

- the date isn't a real calendar date (`2026-02-30` is rejected, not
  silently rolled forward to March)
- the end time equals the start time (zero-length entries are rejected)
- the project/description separator (`:`) is missing, or either side is
  empty
- the entry overlaps another entry, including one on the next day
  (touching is fine: `09:00-12:00` followed by `12:00-13:00` is not an
  overlap)

A time range whose end is earlier than its start, like `22:00-06:00`, is
read as a shift that crosses midnight rather than an error. Its hours are
counted against the day it started on, and it's checked for overlaps
against the following day's entries too.

## Library usage

```go
import (
	"os"

	timesheet "github.com/johny380/timesheet-fmt"
)

func main() {
	f, _ := os.Open("week.txt")
	defer f.Close()

	sheet, err := timesheet.Parse(f)
	if err != nil {
		// err is a *timesheet.ParseError with a line number when the
		// problem is a single bad line, or a plain error for a
		// cross-entry problem like an overlap.
		panic(err)
	}

	os.Stdout.WriteString(timesheet.Format(sheet))
}
```

## CLI

```
go run ./cmd/tsfmt week.txt
```

prints:

```
2026-09-01
  09:00-12:30  acme-corp  environment setup
  13:00-17:30  acme-corp  implement parser
  daily total  8h00m

2026-09-02
  09:00-17:00  internal   planning
  daily total  8h00m

total  16h00m
```

or, on a bad file, a message on stderr pointing at the offending line
and a non-zero exit code.

With `-strict`, it also prints a warning to stderr for every gap between
two entries on the same day (a day's first entry starting late, or the
whole day being empty, isn't a gap - only idle time between logged
entries is):

```
go run ./cmd/tsfmt -strict week.txt
```

```
warning: 2026-09-01: 1h00m gap between 12:00 and 13:00
```

## Status

Early. The parser and pretty printer work and are covered by tests, but
the format and CLI are both minimal.
