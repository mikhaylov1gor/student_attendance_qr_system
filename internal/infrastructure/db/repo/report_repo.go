package repo

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"attendance/internal/domain/policy"
	"attendance/internal/domain/report"
	"attendance/internal/infrastructure/db/txctx"
)

// ReportRepo — плоские SELECT'ы для отчётов. В отличие от agregate-репо,
// здесь нет доменной модели сущностей — только денормализованные строки
// для writer'ов xlsx/csv.
type ReportRepo struct{ db *gorm.DB }

func NewReportRepo(db *gorm.DB) *ReportRepo { return &ReportRepo{db: db} }

var _ report.Repository = (*ReportRepo)(nil)

func (r *ReportRepo) dbx(ctx context.Context) *gorm.DB { return txctx.DBX(ctx, r.db) }

// Query строит SELECT по attendance_records × users × sessions × courses
// и подкачивает пачкой security_check_results (одним IN-запросом).
//
// Валидацию Filter (ровно один основной фильтр) делает сервис.
func (r *ReportRepo) Query(ctx context.Context, f report.Filter) ([]report.RawRow, error) {
	// Базовый JOIN. Email берём из users; ФИО — в зашифрованном виде.
	// ISO-строки timestamptz отдаём через `to_char` в Postgres, чтобы не
	// таскать time.Time туда-сюда: формат стабильный, парсится сервисом.
	sql := `
SELECT
    ar.id::text                                   AS attendance_id,
    ar.session_id::text                           AS session_id,
    u.email                                       AS student_email,
    u.full_name_ciphertext                        AS student_full_name_ciphertext,
    u.full_name_nonce                             AS student_full_name_nonce,
    c.name                                        AS course_name,
    c.code                                        AS course_code,
    to_char(s.starts_at AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS"Z"')         AS session_starts_at,
    to_char(s.ends_at   AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS"Z"')         AS session_ends_at,
    to_char(ar.submitted_at AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')      AS submitted_at,
    ar.preliminary_status::text                   AS preliminary_status,
    ar.final_status::text                         AS final_status
FROM attendance_records ar
JOIN users     u ON u.id = ar.student_id
JOIN sessions  s ON s.id = ar.session_id
JOIN courses   c ON c.id = s.course_id
`
	args := []any{}
	where := []string{}

	switch {
	case f.SessionID != nil:
		where = append(where, "ar.session_id = ?")
		args = append(args, *f.SessionID)
	case f.CourseID != nil:
		where = append(where, "s.course_id = ?")
		args = append(args, *f.CourseID)
	case f.GroupID != nil:
		sql += "JOIN session_groups sg ON sg.session_id = s.id\n"
		where = append(where, "sg.group_id = ?")
		args = append(args, *f.GroupID)
	default:
		return nil, fmt.Errorf("report repo: one of session/group/course required")
	}

	if f.TeacherID != nil {
		where = append(where, "s.teacher_id = ?")
		args = append(args, *f.TeacherID)
	}
	if f.From != nil {
		where = append(where, "s.starts_at >= ?")
		args = append(args, *f.From)
	}
	if f.To != nil {
		where = append(where, "s.starts_at <= ?")
		args = append(args, *f.To)
	}

	sql += "WHERE " + strings.Join(where, " AND ") + "\n"
	sql += "ORDER BY s.starts_at DESC, u.email ASC"

	type rawRow struct {
		AttendanceID              string
		SessionID                 string
		StudentEmail              string
		StudentFullNameCiphertext []byte
		StudentFullNameNonce      []byte
		CourseName                string
		CourseCode                string
		SessionStartsAt           string
		SessionEndsAt             string
		SubmittedAt               string
		PreliminaryStatus         string
		FinalStatus               *string
	}

	var rows []rawRow
	if err := r.dbx(ctx).Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("report query main: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}

	// Батчем тянем checks для всех attendance_id.
	// Gorm Raw с плейсхолдером `?` не разворачивает []string в Postgres-массив,
	// поэтому собираем IN-список вручную. Injection не грозит: ids пришли из
	// uuid-колонки БД, не от клиента.
	placeholders := make([]string, len(rows))
	checkArgs := make([]any, len(rows))
	for i, row := range rows {
		placeholders[i] = "?"
		checkArgs[i] = row.AttendanceID
	}
	checksSQL := `SELECT attendance_id::text, mechanism, status::text AS status
	 FROM security_check_results
	 WHERE attendance_id::text IN (` + strings.Join(placeholders, ",") + `)`

	type checkRow struct {
		AttendanceID string
		Mechanism    string
		Status       string
	}
	var checks []checkRow
	if err := r.dbx(ctx).Raw(checksSQL, checkArgs...).Scan(&checks).Error; err != nil {
		return nil, fmt.Errorf("report query checks: %w", err)
	}

	// Индексируем по attendance_id → mechanism → status.
	byID := make(map[string]map[string]string, len(rows))
	for _, c := range checks {
		m, ok := byID[c.AttendanceID]
		if !ok {
			m = make(map[string]string, 3)
			byID[c.AttendanceID] = m
		}
		m[c.Mechanism] = c.Status
	}

	out := make([]report.RawRow, 0, len(rows))
	for _, row := range rows {
		m := byID[row.AttendanceID]
		out = append(out, report.RawRow{
			AttendanceID:              row.AttendanceID,
			SessionID:                 row.SessionID,
			StudentEmail:              row.StudentEmail,
			StudentFullNameCiphertext: row.StudentFullNameCiphertext,
			StudentFullNameNonce:      row.StudentFullNameNonce,
			CourseName:                row.CourseName,
			CourseCode:                row.CourseCode,
			SessionStartsAt:           row.SessionStartsAt,
			SessionEndsAt:             row.SessionEndsAt,
			SubmittedAt:               row.SubmittedAt,
			PreliminaryStatus:         row.PreliminaryStatus,
			FinalStatus:               row.FinalStatus,
			QRTTLStatus:               m[policy.MechanismQRTTL],
			GeoStatus:                 m[policy.MechanismGeo],
			WiFiStatus:                m[policy.MechanismWiFi],
		})
	}
	return out, nil
}
