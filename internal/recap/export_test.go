package recap

import (
	"strconv"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func sampleDetail() Detail {
	return Detail{
		Recap: Recap{ID: 7, Mode: "custom", Status: "final"},
		Workers: []WorkerLine{
			{
				WorkerID: 1,
				Name:     "Budi",
				Role:     "Tukang Batu",
				Quarters: 22,
				Gross:    825_000,
				Advances: 100_000,
				Extras:   50_000,
				Net:      775_000,
				Paid:     true,
				Attendance: []AttendanceItem{
					{ID: 1, Date: "2026-01-06", Quarters: 4, Earned: 150_000},
					{ID: 2, Date: "2026-01-11", Quarters: 2, Earned: 75_000},
				},
				ExtraFees: []ExtraFee{{ID: 1, WorkerID: 1, Amount: 50_000, Description: "Lembur"}},
			},
		},
	}
}

func TestWhatsAppText(t *testing.T) {
	text := WhatsAppText(sampleDetail().Workers[0])

	checks := []string{
		"*REKAP GAJI*",
		"Nama: Budi (Tukang Batu)",
		"Periode: 06 Jan 2026 - 11 Jan 2026",
		"Kehadiran: 5,5 hari",
		"Gaji kotor: Rp 825.000",
		"Kas bon: - Rp 100.000",
		"Tambahan: + Rp 50.000 (Lembur)",
		"*Total diterima: Rp 775.000*",
	}
	for _, expected := range checks {
		if !strings.Contains(text, expected) {
			t.Fatalf("message missing %q:\n%s", expected, text)
		}
	}
}

func TestWhatsAppTextOmitsEmptyLines(t *testing.T) {
	line := sampleDetail().Workers[0]
	line.Advances = 0
	line.Extras = 0
	line.ExtraFees = nil

	text := WhatsAppText(line)
	if strings.Contains(text, "Kas bon") {
		t.Fatalf("zero kas bon must be omitted:\n%s", text)
	}
	if strings.Contains(text, "Tambahan") {
		t.Fatalf("zero extras must be omitted:\n%s", text)
	}
}

func TestExcelExport(t *testing.T) {
	buffer, err := Excel(sampleDetail())
	if err != nil {
		t.Fatalf("Excel: %v", err)
	}
	if buffer.Len() == 0 {
		t.Fatal("workbook is empty")
	}

	file, err := excelize.OpenReader(buffer)
	if err != nil {
		t.Fatalf("open workbook: %v", err)
	}
	defer file.Close()

	name, err := file.GetCellValue("Rekap", "B2")
	if err != nil || name != "Budi" {
		t.Fatalf("B2 = %q, err = %v", name, err)
	}

	raw, err := file.GetCellValue("Rekap", "E2", excelize.Options{RawCellValue: true})
	if err != nil {
		t.Fatalf("read gross: %v", err)
	}
	gross, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		t.Fatalf("gross %q is not numeric: %v", raw, err)
	}
	if gross != 825_000 {
		t.Fatalf("gross = %d, want 825000", gross)
	}

	date, err := file.GetCellValue("Detail Harian", "B2")
	if err != nil || date != "2026-01-06" {
		t.Fatalf("detail date = %q, err = %v", date, err)
	}
}

func TestFormatHelpers(t *testing.T) {
	if got := formatRupiah(1_250_000); got != "Rp 1.250.000" {
		t.Fatalf("formatRupiah = %q", got)
	}
	if got := formatRupiah(-50_000); got != "-Rp 50.000" {
		t.Fatalf("negative formatRupiah = %q", got)
	}
	if got := formatDays(21); got != "5,25" {
		t.Fatalf("formatDays 21 = %q", got)
	}
	if got := formatDays(23); got != "5,75" {
		t.Fatalf("formatDays 23 = %q", got)
	}
}

func TestAllWhatsAppTextIsCompactWithOverallTotal(t *testing.T) {
	detail := sampleDetail()
	detail.PeriodStart = "2026-01-06"
	detail.PeriodEnd = "2026-01-11"
	detail.Totals = Totals{Gross: 1_325_000, Advances: 150_000, Extras: 50_000, Net: 1_275_000}
	detail.Workers = append(detail.Workers, WorkerLine{
		WorkerID: 2,
		Name:     "Andi",
		Role:     "Tukang Kayu",
		Net:      500_000,
	})

	text := AllWhatsAppText(detail)

	for _, expected := range []string{
		"Periode: 06 Jan 2026 - 11 Jan 2026",
		"Budi: Rp 775.000",
		"Andi: Rp 500.000",
		"Gaji kotor: Rp 1.325.000",
		"Kas bon: - Rp 150.000",
		"Tambahan: + Rp 50.000",
		"*Total dibayar: Rp 1.275.000*",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("message missing %q:\n%s", expected, text)
		}
	}

	if strings.Count(text, "Kehadiran:") != 0 {
		t.Fatalf("all-workers message must not repeat per-worker breakdown:\n%s", text)
	}
}
