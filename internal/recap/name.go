package recap

import (
	"fmt"
	"time"

	"situkang/internal/clock"
)

var weekdayNames = [...]string{
	"Minggu",
	"Senin",
	"Selasa",
	"Rabu",
	"Kamis",
	"Jumat",
	"Sabtu",
}

var longMonthNames = [...]string{
	"",
	"Januari",
	"Februari",
	"Maret",
	"April",
	"Mei",
	"Juni",
	"Juli",
	"Agustus",
	"September",
	"Oktober",
	"November",
	"Desember",
}

// jakartaZone is a fixed UTC+7 zone. Jakarta does not observe daylight saving
// time, so a fixed zone keeps the name helper independent from tzdata.
var jakartaZone = time.FixedZone("Asia/Jakarta", 7*60*60)

// recapName renders the recap creation timestamp in Asia/Jakarta, for example
// "Rekap Sabtu, 10 Oktober 2026".
func recapName(createdAt string) string {
	t, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		parsed, parseErr := clock.ParseDate(createdAt)
		if parseErr != nil {
			return "Rekap"
		}
		t = parsed
	}

	local := t.In(jakartaZone)
	return fmt.Sprintf("Rekap %s, %d %s %d",
		weekdayNames[local.Weekday()], local.Day(), longMonthNames[local.Month()], local.Year())
}
