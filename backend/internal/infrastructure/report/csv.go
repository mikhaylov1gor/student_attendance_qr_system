// Package report — форматтеры XLSX и CSV для отчётов.
// На вход берут []domain.ReportRow, на выход пишут в io.Writer — стримятся
// прямо в http.ResponseWriter.
package report

import (
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"attendance/internal/domain/report"
)

// BOM — UTF-8 byte order mark. Excel на Windows без BOM'а открывает CSV в
// CP1251 и ломает кириллицу. С BOM'ом — корректно распознаёт как UTF-8.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// Headers — общие заголовки для xlsx и csv. Порядок фиксирован.
var Headers = []string{
	"ФИО студента",
	"Email",
	"Курс",
	"Код курса",
	"Дата занятия",
	"Подано в",
	"Preliminary",
	"Final",
	"Effective",
	"qr_ttl",
	"geo",
	"wifi",
}

// WriteCSV пишет отчёт в w. Первым идёт UTF-8 BOM, затем строки с разделителем `;`
// (Excel по умолчанию ждёт `;` в русской локали, а не `,`).
func WriteCSV(w io.Writer, rows []report.ReportRow) error {
	if _, err := w.Write(utf8BOM); err != nil {
		return fmt.Errorf("csv write BOM: %w", err)
	}
	cw := csv.NewWriter(w)
	cw.Comma = ';'
	if err := cw.Write(Headers); err != nil {
		return fmt.Errorf("csv header: %w", err)
	}
	for _, r := range rows {
		rec := rowToStrings(r)
		if err := cw.Write(rec); err != nil {
			return fmt.Errorf("csv row: %w", err)
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return fmt.Errorf("csv flush: %w", err)
	}
	return nil
}

// rowToStrings — общий маппер для CSV и XLSX (xlsx использует []any,
// но строки подходят без касти).
func rowToStrings(r report.ReportRow) []string {
	return []string{
		r.StudentFullName,
		r.StudentEmail,
		r.CourseName,
		r.CourseCode,
		r.SessionStartsAt.Format(time.RFC3339),
		r.SubmittedAt.Format(time.RFC3339),
		r.PreliminaryStatus,
		r.FinalStatus,
		r.EffectiveStatus,
		r.QRTTLStatus,
		r.GeoStatus,
		r.WiFiStatus,
	}
}
