// Package attendance — use case приёма отметки студента.
//
// Submit-пайплайн (точка сложности #2):
//  1. decode QR-токена → {session_id, counter, issued_at}
//  2. load session (status=active, время ∈ [starts_at, ends_at])
//  3. verify HMAC через QRTokenCodec с session.QRSecret
//  4. pre-check uniqueness (session_id, student_id)
//  5. engine.Evaluate(policy.Mechanisms, CheckInput) → []CheckResult
//  6. DerivePreliminaryStatus(results) → accepted | needs_review
//  7. Tx.Run: attendance.Submit(record + checks) + audit.Append
//  8. hub.Broadcast — teacher видит live
//
// Ошибки на шагах 1–3 → 400/422 с доменной ErrInvalidQRToken.
// Шаг 4 → 409 ErrAlreadySubmitted.
package attendance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	appaudit "attendance/internal/application/audit"
	"attendance/internal/application/hub"
	"attendance/internal/domain"
	"attendance/internal/domain/attendance"
	"attendance/internal/domain/audit"
	"attendance/internal/domain/catalog"
	"attendance/internal/domain/policy"
	"attendance/internal/domain/session"
	"attendance/internal/platform/authctx"
	"attendance/internal/platform/requestmeta"
)

// Deps — всё, что нужно сервису.
type Deps struct {
	Attendance attendance.Repository
	Sessions   session.Repository
	Policies   policy.Repository
	Classrooms catalog.ClassroomRepository
	Engine     *policy.Engine
	Codec      domain.QRTokenCodec
	Tx         domain.TxRunner
	Audit      *appaudit.Service
	Hub        *hub.Hub
	Clock      domain.Clock
}

type Service struct{ Deps }

func NewService(d Deps) *Service { return &Service{Deps: d} }

// SubmitInput — то, что приходит с мобилки.
type SubmitInput struct {
	QRToken string
	// Геолокация клиента — опциональна.
	GeoLat *float64
	GeoLng *float64
	// Wi-Fi BSSID — опционален.
	BSSID *string
	// ClientTime — время по часам клиента (forensics). Если 0 — подставим серверное.
	ClientTime time.Time
}

// SubmitResult — возвращаем handler'у для ответа клиенту.
type SubmitResult struct {
	Record attendance.Record
	Checks []attendance.CheckResult
}

// Submit выполняет полный пайплайн.
func (s *Service) Submit(ctx context.Context, in SubmitInput) (SubmitResult, error) {
	principal, err := authctx.Require(ctx)
	if err != nil {
		return SubmitResult{}, err
	}

	// 1. Парсим токен без верификации (чтобы узнать session_id).
	token, err := s.decodeUnverified(in.QRToken)
	if err != nil {
		return SubmitResult{}, attendance.ErrInvalidQRToken
	}

	// 2. Грузим сессию.
	sess, err := s.Sessions.GetByID(ctx, token.SessionID)
	if err != nil {
		if errors.Is(err, session.ErrNotFound) {
			return SubmitResult{}, attendance.ErrInvalidQRToken
		}
		return SubmitResult{}, fmt.Errorf("submit: load session: %w", err)
	}
	now := s.Clock.Now(ctx)
	if !sess.IsAcceptingAttendance(now) {
		return SubmitResult{}, session.ErrNotAcceptingAttendance
	}

	// 3. Полноценная верификация HMAC с session.QRSecret.
	if _, err := s.Codec.Decode(sess.QRSecret, in.QRToken); err != nil {
		return SubmitResult{}, attendance.ErrInvalidQRToken
	}

	// 4. Pre-check уникальности (финальная защита — unique-index в БД).
	exists, err := s.Attendance.ExistsForSessionStudent(ctx, sess.ID, principal.UserID)
	if err != nil {
		return SubmitResult{}, fmt.Errorf("submit: uniq check: %w", err)
	}
	if exists {
		return SubmitResult{}, attendance.ErrAlreadySubmitted
	}

	// 5. Готовим CheckInput для engine.
	pol, err := s.Policies.GetByID(ctx, sess.SecurityPolicyID)
	if err != nil {
		return SubmitResult{}, fmt.Errorf("submit: load policy: %w", err)
	}

	input := policy.CheckInput{
		SessionID:      sess.ID,
		ClassroomID:    sess.ClassroomID,
		TokenCounter:   token.Counter,
		CurrentCounter: sess.QRCounter,
		TokenIssuedAt:  token.IssuedAt,
		ClientGeoLat:   in.GeoLat,
		ClientGeoLng:   in.GeoLng,
		ClientBSSID:    in.BSSID,
		ClientTime:     in.ClientTime,
	}
	if sess.ClassroomID != nil {
		cls, err := s.Classrooms.GetByID(ctx, *sess.ClassroomID)
		if err != nil {
			return SubmitResult{}, fmt.Errorf("submit: load classroom: %w", err)
		}
		input.ClassroomLat = cls.Latitude
		input.ClassroomLng = cls.Longitude
		input.ClassroomRadius = cls.RadiusMeters
		input.AllowedBSSIDs = cls.AllowedBSSIDs
	}

	results, err := s.Engine.Evaluate(ctx, pol.Mechanisms, input)
	if err != nil {
		return SubmitResult{}, fmt.Errorf("submit: engine: %w", err)
	}

	// 6. Derive preliminary_status (accepted | needs_review).
	prelim := attendance.DerivePreliminaryStatus(results)

	// 7. Собираем record + check results.
	recordID := uuid.New()
	record := attendance.Record{
		ID:                recordID,
		SessionID:         sess.ID,
		StudentID:         principal.UserID,
		SubmittedAt:       now,
		SubmittedQRToken:  in.QRToken,
		PreliminaryStatus: prelim,
	}
	checks := make([]attendance.CheckResult, 0, len(results))
	for _, r := range results {
		checks = append(checks, attendance.CheckResult{
			ID:           uuid.New(),
			AttendanceID: recordID,
			Mechanism:    r.Mechanism,
			Status:       attendance.CheckStatus(r.Status), // строковые значения совпадают
			Details:      r.Details,
			CheckedAt:    now,
		})
	}

	// Транзакция: persist + audit.
	err = s.Tx.Run(ctx, func(txCtx context.Context) error {
		if err := s.Attendance.Submit(txCtx, record, checks); err != nil {
			return err
		}
		return s.auditAppend(txCtx, audit.Entry{
			ActorID:    &principal.UserID,
			ActorRole:  string(principal.Role),
			Action:     audit.ActionAttendanceSubmitted,
			EntityType: "attendance",
			EntityID:   record.ID.String(),
			Payload: map[string]any{
				"attendance_id":      record.ID.String(),
				"session_id":         sess.ID.String(),
				"student_id":         principal.UserID.String(),
				"preliminary_status": string(prelim),
				"checks":             summarizeChecks(results),
			},
		})
	})
	if err != nil {
		return SubmitResult{}, err
	}

	// 8. Broadcast teacher'у (после commit — чтобы не гонять события отката).
	s.broadcastAttendance(ctx, sess.ID, record, checks)

	return SubmitResult{Record: record, Checks: checks}, nil
}

// decodeUnverified читает payload из base64url без проверки HMAC (для определения
// session_id). Верификация подписи — на шаге 3 с сессионным secret'ом.
// Здесь используем тот же Codec с заведомо неверным secret — просто ловим
// парсинг-ошибку как признак bad-format. Для отделения session_id от HMAC-ошибки
// делаем трюк: пытаемся декодировать с нулевым secret'ом; если не получается
// из-за формата — сразу ошибка, если из-за HMAC — парсим вручную. Проще —
// обойти через «decode + игнорить ошибку подписи», но это приходится делать
// вручную. Делаем безопасно: декодируем base64 и вырезаем первые 16 байт как
// session_id.
func (s *Service) decodeUnverified(token string) (domain.QRToken, error) {
	// Проверим формат минимально (length + base64). Если не base64url 86 символов — invalid.
	// Используем full decode через Codec с заведомо мусорным secret — если ошибка
	// НЕ про подпись, а про формат, это нам и надо; иначе — достанем session_id
	// через повторный ручной парсинг. Для простоты и стабильности — ручной разбор.
	return parseSessionFromToken(token)
}

func (s *Service) broadcastAttendance(
	ctx context.Context,
	sessionID uuid.UUID,
	r attendance.Record,
	checks []attendance.CheckResult,
) {
	payload := map[string]any{
		"type":               "attendance",
		"attendance_id":      r.ID.String(),
		"session_id":         sessionID.String(),
		"student_id":         r.StudentID.String(),
		"submitted_at":       r.SubmittedAt,
		"preliminary_status": string(r.PreliminaryStatus),
		"checks":             summarizeAttendanceChecks(checks),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	s.Hub.Broadcast(ctx, sessionID, data)
}

func (s *Service) auditAppend(ctx context.Context, e audit.Entry) error {
	if s.Audit == nil {
		return nil
	}
	meta := requestmeta.From(ctx)
	e.IPAddress = meta.RemoteIP
	e.UserAgent = meta.UserAgent
	_, err := s.Audit.Append(ctx, e)
	return err
}

func summarizeChecks(rs []policy.CheckResult) []map[string]any {
	out := make([]map[string]any, 0, len(rs))
	for _, r := range rs {
		out = append(out, map[string]any{
			"mechanism": r.Mechanism,
			"status":    string(r.Status),
		})
	}
	return out
}

func summarizeAttendanceChecks(cs []attendance.CheckResult) []map[string]any {
	out := make([]map[string]any, 0, len(cs))
	for _, c := range cs {
		out = append(out, map[string]any{
			"mechanism": c.Mechanism,
			"status":    string(c.Status),
			"details":   c.Details,
		})
	}
	return out
}
