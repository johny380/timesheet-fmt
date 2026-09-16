// Package timesheet parses a plain-text time sheet format:
//
//	2026-09-01 09:00-12:30 acme-corp: environment setup
//	2026-09-01 13:00-17:30 acme-corp: implement parser
//
// Each line is a date, a time range, and a "project: description" pair.
// Blank lines and lines starting with "#" are ignored. A time range whose
// end is earlier than its start (e.g. 22:00-06:00) is read as a shift that
// crosses midnight and ends on the following day; its hours are counted
// against the day it started on.
package timesheet

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// Entry is one logged block of work. End <= Start means the shift crosses
// midnight and ends on the day after Day.
type Entry struct {
	Day         time.Time // date only, time-of-day is zero
	Start       int       // minutes since midnight on Day
	End         int       // minutes since midnight on Day, or the next day
	Project     string
	Description string
}

// Minutes returns how long the entry lasted. A shift that crosses midnight
// (End <= Start) is assumed to run into the following day rather than
// wrapping past it a second time.
func (e Entry) Minutes() int {
	if e.End <= e.Start {
		return (24*60 - e.Start) + e.End
	}
	return e.End - e.Start
}

// startTime and endTime give the entry's absolute start and end, so
// overlap checks work the same whether or not a shift crosses midnight.
func (e Entry) startTime() time.Time {
	return e.Day.Add(time.Duration(e.Start) * time.Minute)
}

func (e Entry) endTime() time.Time {
	t := e.Day.Add(time.Duration(e.End) * time.Minute)
	if e.End <= e.Start {
		t = t.AddDate(0, 0, 1)
	}
	return t
}

// Sheet is a parsed, validated collection of entries, sorted by day then
// start time.
type Sheet struct {
	Entries []Entry
}

// TotalMinutes sums the duration of every entry in the sheet.
func (s *Sheet) TotalMinutes() int {
	total := 0
	for _, e := range s.Entries {
		total += e.Minutes()
	}
	return total
}

// ParseError reports the input line a parse failure happened on.
type ParseError struct {
	Line int
	Msg  string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d: %s", e.Line, e.Msg)
}

// Parse reads a time sheet, validating every entry and rejecting the file
// as a whole if any entries on the same day overlap.
func Parse(r io.Reader) (*Sheet, error) {
	scanner := bufio.NewScanner(r)
	var entries []Entry
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		entry, err := parseLine(text)
		if err != nil {
			return nil, &ParseError{Line: line, Msg: err.Error()}
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if !entries[i].Day.Equal(entries[j].Day) {
			return entries[i].Day.Before(entries[j].Day)
		}
		return entries[i].Start < entries[j].Start
	})

	if err := checkOverlaps(entries); err != nil {
		return nil, err
	}

	return &Sheet{Entries: entries}, nil
}

func parseLine(text string) (Entry, error) {
	fields := strings.SplitN(text, " ", 3)
	if len(fields) < 3 {
		return Entry{}, fmt.Errorf("expected \"date time-range project: description\", got %q", text)
	}
	dateStr, rangeStr, rest := fields[0], fields[1], fields[2]

	// time.Parse silently normalizes out-of-range days (2026-02-30 becomes
	// 2026-03-02), so the round trip through Format is what actually
	// catches a bad calendar date.
	day, err := time.Parse("2006-01-02", dateStr)
	if err != nil || day.Format("2006-01-02") != dateStr {
		return Entry{}, fmt.Errorf("invalid date %q", dateStr)
	}

	start, end, err := parseRange(rangeStr)
	if err != nil {
		return Entry{}, err
	}

	colon := strings.Index(rest, ":")
	if colon < 0 {
		return Entry{}, fmt.Errorf("missing \":\" between project and description in %q", rest)
	}
	project := strings.TrimSpace(rest[:colon])
	description := strings.TrimSpace(rest[colon+1:])
	if project == "" {
		return Entry{}, fmt.Errorf("empty project name")
	}
	if description == "" {
		return Entry{}, fmt.Errorf("empty description")
	}

	return Entry{
		Day:         day,
		Start:       start,
		End:         end,
		Project:     project,
		Description: description,
	}, nil
}

func parseRange(s string) (start, end int, err error) {
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid time range %q, want HH:MM-HH:MM", s)
	}
	start, err = parseClock(parts[0])
	if err != nil {
		return 0, 0, err
	}
	end, err = parseClock(parts[1])
	if err != nil {
		return 0, 0, err
	}
	// end == start is always zero duration, which is always an error.
	// end < start isn't an error: it's read as a shift that crosses
	// midnight and ends the following day.
	if end == start {
		return 0, 0, fmt.Errorf("end time must be after start time in %q", s)
	}
	return start, end, nil
}

func parseClock(s string) (int, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, fmt.Errorf("invalid time %q: %w", s, err)
	}
	return t.Hour()*60 + t.Minute(), nil
}

// checkOverlaps assumes entries are sorted by day then start time, which
// is also their absolute chronological order. Under that ordering it's
// enough to compare each entry with its immediate predecessor: if no
// adjacent pair overlaps, transitivity of the sort guarantees no pair
// anywhere overlaps either. Comparing absolute start/end rather than raw
// minutes-since-midnight means a shift that crosses midnight is checked
// against the following day's entries too.
func checkOverlaps(entries []Entry) error {
	for i := 1; i < len(entries); i++ {
		prev, cur := entries[i-1], entries[i]
		if !cur.startTime().Before(prev.endTime()) {
			continue
		}
		if prev.Day.Equal(cur.Day) {
			return fmt.Errorf("overlapping entries on %s: %s-%s and %s-%s",
				cur.Day.Format("2006-01-02"),
				formatClock(prev.Start), formatClock(prev.End),
				formatClock(cur.Start), formatClock(cur.End))
		}
		return fmt.Errorf("overlapping entries: %s %s-%s and %s %s-%s",
			prev.Day.Format("2006-01-02"), formatClock(prev.Start), formatClock(prev.End),
			cur.Day.Format("2006-01-02"), formatClock(cur.Start), formatClock(cur.End))
	}
	return nil
}

func formatClock(m int) string {
	return fmt.Sprintf("%02d:%02d", m/60, m%60)
}

// Gap describes idle time between two entries that fall on the same day.
// Entries on different days are never reported as a gap: the time between
// one day's last entry and the next day's first is normal, not a hole in
// the schedule.
type Gap struct {
	Day       time.Time
	PrevEnd   int // minutes since midnight, end of the entry before the gap
	NextStart int // minutes since midnight, start of the entry after the gap
	Minutes   int
}

func (g Gap) String() string {
	return fmt.Sprintf("%s: %s gap between %s and %s",
		g.Day.Format("2006-01-02"), formatMinutes(g.Minutes), formatClock(g.PrevEnd), formatClock(g.NextStart))
}

// FindGaps reports every idle stretch between consecutive same-day entries
// in s. It assumes s.Entries is sorted and overlap-free, which is true for
// anything returned by Parse.
func FindGaps(s *Sheet) []Gap {
	var gaps []Gap
	for i := 1; i < len(s.Entries); i++ {
		prev, cur := s.Entries[i-1], s.Entries[i]
		if !prev.Day.Equal(cur.Day) {
			continue
		}
		idle := int(cur.startTime().Sub(prev.endTime()).Minutes())
		if idle <= 0 {
			continue
		}
		gaps = append(gaps, Gap{
			Day:       cur.Day,
			PrevEnd:   prev.End,
			NextStart: cur.Start,
			Minutes:   idle,
		})
	}
	return gaps
}
