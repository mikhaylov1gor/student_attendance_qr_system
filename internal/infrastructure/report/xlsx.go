package report

import (
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"

	"attendance/internal/domain/report"
)

// WriteXLSX генерирует xlsx-файл и пишет его в w.
//
// Формат: один лист "Attendance", первая строка — заголовки жирным,
// ширина колонок — fixed (читаемо без ручной настройки).
func WriteXLSX(w io.Writer, rows []report.ReportRow) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Attendance"
	idx, err := f.NewSheet(sheet)
	if err != nil {
		return fmt.Errorf("xlsx new sheet: %w", err)
	}
	// Дефолтный Sheet1 убираем.
	if err := f.DeleteSheet("Sheet1"); err != nil {
		return fmt.Errorf("xlsx delete default sheet: %w", err)
	}
	f.SetActiveSheet(idx)

	// Заголовки.
	for col, h := range Headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		if err := f.SetCellStr(sheet, cell, h); err != nil {
			return fmt.Errorf("xlsx header: %w", err)
		}
	}
	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return fmt.Errorf("xlsx style: %w", err)
	}
	lastCol, _ := excelize.ColumnNumberToName(len(Headers))
	if err := f.SetCellStyle(sheet, "A1", lastCol+"1", bold); err != nil {
		return fmt.Errorf("xlsx style header: %w", err)
	}

	// Разумные ширины колонок.
	widths := map[string]float64{
		"A": 32, // ФИО
		"B": 28, // Email
		"C": 26, // Курс
		"D": 12, // Код
		"E": 22, // Дата занятия
		"F": 22, // Submitted at
		"G": 14, // Preliminary
		"H": 14, // Final
		"I": 14, // Effective
		"J": 10, // qr_ttl
		"K": 10, // geo
		"L": 10, // wifi
	}
	for col, w := range widths {
		if err := f.SetColWidth(sheet, col, col, w); err != nil {
			return fmt.Errorf("xlsx col width: %w", err)
		}
	}

	// Данные.
	for i, r := range rows {
		rowNum := i + 2 // 1 — заголовки
		strs := rowToStrings(r)
		for col, v := range strs {
			cell, _ := excelize.CoordinatesToCellName(col+1, rowNum)
			if err := f.SetCellStr(sheet, cell, v); err != nil {
				return fmt.Errorf("xlsx cell [%d,%d]: %w", col+1, rowNum, err)
			}
		}
	}

	if err := f.Write(w); err != nil {
		return fmt.Errorf("xlsx write: %w", err)
	}
	return nil
}
