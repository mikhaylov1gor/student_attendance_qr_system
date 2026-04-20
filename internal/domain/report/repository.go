package report

import "context"

// Repository — порт для report-репозитория.
// Возвращает плоские строки с шифрованным ФИО (Ciphertext/Nonce) —
// сервис расшифровывает через FieldEncryptor.
type Repository interface {
	Query(ctx context.Context, f Filter) ([]RawRow, error)
}

// RawRow — то, что приходит из БД: шифрованное ФИО + остальное plaintext.
// Сервис делает маппинг RawRow → ReportRow с расшифровкой ФИО.
type RawRow struct {
	AttendanceID string

	SessionID string

	StudentEmail              string
	StudentFullNameCiphertext []byte
	StudentFullNameNonce      []byte

	CourseName string
	CourseCode string

	SessionStartsAt string // ISO; парсится сервисом
	SessionEndsAt   string

	SubmittedAt       string
	PreliminaryStatus string
	FinalStatus       *string

	// Развёрнуто по механизмам: если отсутствует — "".
	QRTTLStatus string
	GeoStatus   string
	WiFiStatus  string
}
