package recap

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"

	"situkang/internal/clock"
)

var monthNames = [...]string{
	"Jan", "Feb", "Mar", "Apr", "Mei", "Jun",
	"Jul", "Agu", "Sep", "Okt", "Nov", "Des",
}

// WhatsAppText renders the per-worker recap message.
func WhatsAppText(line WorkerLine) string {
	var b strings.Builder

	fmt.Fprintf(&b, "*REKAP GAJI*\n")
	fmt.Fprintf(&b, "Nama: %s (%s)\n", line.Name, line.Role)
	fmt.Fprintf(&b, "Periode: %s\n\n", periodLabel(line))
	fmt.Fprintf(&b, "Kehadiran: %s hari\n", formatDays(line.Quarters))
	fmt.Fprintf(&b, "Gaji kotor: %s\n", formatRupiah(line.Gross))

	if line.Advances > 0 {
		fmt.Fprintf(&b, "Kas bon: - %s\n", formatRupiah(line.Advances))
	}
	if line.Extras != 0 {
		for _, fee := range line.ExtraFees {
			sign := "+"
			if fee.Amount < 0 {
				sign = "-"
			}
			fmt.Fprintf(&b, "Tambahan: %s %s (%s)\n", sign, formatRupiah(abs(fee.Amount)), fee.Description)
		}
	}

	fmt.Fprintf(&b, "*Total diterima: %s*", formatRupiah(line.Net))
	return b.String()
}

// AllWhatsAppText joins every worker message with a blank line.
func AllWhatsAppText(detail Detail) string {
	parts := make([]string, 0, len(detail.Workers))
	for _, line := range detail.Workers {
		parts = append(parts, WhatsAppText(line))
	}
	return strings.Join(parts, "\n\n")
}

func periodLabel(line WorkerLine) string {
	first, last := "", ""
	track := func(date string) {
		if first == "" || date < first {
			first = date
		}
		if last == "" || date > last {
			last = date
		}
	}
	for _, item := range line.Attendance {
		track(item.Date)
	}
	for _, item := range line.CashAdvances {
		track(item.Date)
	}
	if first == "" {
		return "-"
	}
	if first == last {
		return formatDateID(first)
	}
	return formatDateID(first) + " - " + formatDateID(last)
}

// Excel builds the export workbook: one summary sheet plus daily detail.
func Excel(detail Detail) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer f.Close()

	const summary = "Rekap"
	const daily = "Detail Harian"

	summaryIndex, err := f.NewSheet(summary)
	if err != nil {
		return nil, err
	}
	f.SetActiveSheet(summaryIndex)
	if err := f.DeleteSheet("Sheet1"); err != nil {
		return nil, err
	}

	moneyStyle, err := f.NewStyle(&excelize.Style{
		NumFmt:       164,
		CustomNumFmt: stringPtr(`"Rp" #,##0`),
	})
	if err != nil {
		return nil, err
	}
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E5E7EB"}},
	})
	if err != nil {
		return nil, err
	}

	headers := []string{"No", "Nama", "Peran", "Kehadiran (hari)", "Gaji Kotor", "Kas Bon", "Tambahan", "Total", "Dibayar"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(summary, cell, header); err != nil {
			return nil, err
		}
	}
	if err := f.SetCellStyle(summary, "A1", "I1", headerStyle); err != nil {
		return nil, err
	}

	for index, line := range detail.Workers {
		row := index + 2
		values := []any{
			index + 1,
			line.Name,
			line.Role,
			float64(line.Quarters) / 4,
			line.Gross,
			line.Advances,
			line.Extras,
			line.Net,
			map[bool]string{true: "Ya", false: "Tidak"}[line.Paid],
		}
		for column, value := range values {
			cell, _ := excelize.CoordinatesToCellName(column+1, row)
			if err := f.SetCellValue(summary, cell, value); err != nil {
				return nil, err
			}
		}
		if err := f.SetCellStyle(summary, fmt.Sprintf("E%d", row), fmt.Sprintf("H%d", row), moneyStyle); err != nil {
			return nil, err
		}
	}

	_ = f.SetColWidth(summary, "A", "A", 5)
	_ = f.SetColWidth(summary, "B", "B", 22)
	_ = f.SetColWidth(summary, "C", "C", 18)
	_ = f.SetColWidth(summary, "D", "I", 16)

	if _, err := f.NewSheet(daily); err != nil {
		return nil, err
	}
	dailyHeaders := []string{"Nama", "Tanggal", "Bagian Hari", "Upah"}
	for i, header := range dailyHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(daily, cell, header); err != nil {
			return nil, err
		}
	}
	if err := f.SetCellStyle(daily, "A1", "D1", headerStyle); err != nil {
		return nil, err
	}

	row := 2
	for _, line := range detail.Workers {
		for _, item := range line.Attendance {
			values := []any{line.Name, item.Date, formatDays(item.Quarters), item.Earned}
			for column, value := range values {
				cell, _ := excelize.CoordinatesToCellName(column+1, row)
				if err := f.SetCellValue(daily, cell, value); err != nil {
					return nil, err
				}
			}
			if err := f.SetCellStyle(daily, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), moneyStyle); err != nil {
				return nil, err
			}
			row++
		}
	}
	_ = f.SetColWidth(daily, "A", "A", 22)
	_ = f.SetColWidth(daily, "B", "D", 14)

	buffer, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

func formatRupiah(value int64) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	digits := fmt.Sprintf("%d", value)
	var b strings.Builder
	for i, digit := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(digit)
	}
	return sign + "Rp " + b.String()
}

func formatDays(quarters int) string {
	whole := quarters / 4
	remainder := quarters % 4
	switch remainder {
	case 0:
		return fmt.Sprintf("%d", whole)
	case 1:
		return fmt.Sprintf("%d,25", whole)
	case 2:
		return fmt.Sprintf("%d,5", whole)
	default:
		return fmt.Sprintf("%d,75", whole)
	}
}

func formatDateID(value string) string {
	t, err := clock.ParseDate(value)
	if err != nil {
		return value
	}
	return fmt.Sprintf("%02d %s %d", t.Day(), monthNames[t.Month()-1], t.Year())
}

func abs(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

func stringPtr(value string) *string {
	return &value
}
