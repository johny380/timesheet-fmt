package timesheet

import (
	"fmt"
	"strings"
)

// Format renders a sheet grouped by day, with a running total per day and
// a grand total at the end. It assumes s.Entries is already sorted, which
// is true for anything returned by Parse.
func Format(s *Sheet) string {
	if len(s.Entries) == 0 {
		return ""
	}

	projectWidth := 0
	for _, e := range s.Entries {
		if len(e.Project) > projectWidth {
			projectWidth = len(e.Project)
		}
	}

	var b strings.Builder
	grandTotal := 0
	i := 0
	for i < len(s.Entries) {
		day := s.Entries[i].Day
		j := i
		dayTotal := 0
		for j < len(s.Entries) && s.Entries[j].Day.Equal(day) {
			dayTotal += s.Entries[j].Minutes()
			j++
		}

		fmt.Fprintf(&b, "%s\n", day.Format("2006-01-02"))
		for _, e := range s.Entries[i:j] {
			fmt.Fprintf(&b, "  %s-%s  %-*s  %s\n",
				formatClock(e.Start), formatClock(e.End), projectWidth, e.Project, e.Description)
		}
		fmt.Fprintf(&b, "  daily total  %s\n\n", formatMinutes(dayTotal))

		grandTotal += dayTotal
		i = j
	}
	fmt.Fprintf(&b, "total  %s\n", formatMinutes(grandTotal))
	return b.String()
}

func formatMinutes(m int) string {
	return fmt.Sprintf("%dh%02dm", m/60, m%60)
}
