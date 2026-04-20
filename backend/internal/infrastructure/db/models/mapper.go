package models

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/google/uuid"

	"attendance/internal/domain"
	"attendance/internal/domain/attendance"
	"attendance/internal/domain/audit"
	"attendance/internal/domain/catalog"
	"attendance/internal/domain/policy"
	"attendance/internal/domain/session"
	"attendance/internal/domain/user"
)

// ============================================================================
// User mappers. Требуют FieldEncryptor для ФИО.
// ============================================================================

// fullNameEnvelope — формат plaintext'а, который шифруется в bytea.
type fullNameEnvelope struct {
	L string `json:"l"`
	F string `json:"f"`
	M string `json:"m,omitempty"`
}

func FullNameEncrypt(fn user.FullName, enc domain.FieldEncryptor) (ciphertext, nonce []byte, err error) {
	payload, err := json.Marshal(fullNameEnvelope{L: fn.Last, F: fn.First, M: fn.Middle})
	if err != nil {
		return nil, nil, fmt.Errorf("marshal fullname: %w", err)
	}
	return enc.Encrypt(payload)
}

func FullNameDecrypt(ciphertext, nonce []byte, enc domain.FieldEncryptor) (user.FullName, error) {
	plain, err := enc.Decrypt(ciphertext, nonce)
	if err != nil {
		return user.FullName{}, fmt.Errorf("decrypt fullname: %w", err)
	}
	var env fullNameEnvelope
	if err := json.Unmarshal(plain, &env); err != nil {
		return user.FullName{}, fmt.Errorf("unmarshal fullname: %w", err)
	}
	return user.FullName{Last: env.L, First: env.F, Middle: env.M}, nil
}

func UserToModel(u user.User, enc domain.FieldEncryptor) (*UserModel, error) {
	ct, nonce, err := FullNameEncrypt(u.FullName, enc)
	if err != nil {
		return nil, err
	}
	m := &UserModel{
		ID:                 u.ID,
		Email:              u.Email,
		PasswordHash:       u.PasswordHash,
		FullNameCiphertext: ct,
		FullNameNonce:      nonce,
		Role:               string(u.Role),
		CurrentGroupID:     u.CurrentGroupID,
		CreatedAt:          u.CreatedAt,
		DeletedAt:          u.DeletedAt,
	}
	return m, nil
}

func UserFromModel(m UserModel, enc domain.FieldEncryptor) (user.User, error) {
	fn, err := FullNameDecrypt(m.FullNameCiphertext, m.FullNameNonce, enc)
	if err != nil {
		return user.User{}, err
	}
	return user.User{
		ID:             m.ID,
		Email:          m.Email,
		PasswordHash:   m.PasswordHash,
		FullName:       fn,
		Role:           user.Role(m.Role),
		CurrentGroupID: m.CurrentGroupID,
		CreatedAt:      m.CreatedAt,
		DeletedAt:      m.DeletedAt,
	}, nil
}

// ============================================================================
// Catalog mappers
// ============================================================================

func CourseToModel(c catalog.Course) CourseModel {
	return CourseModel{
		ID:        c.ID,
		Name:      c.Name,
		Code:      c.Code,
		CreatedAt: c.CreatedAt,
		DeletedAt: c.DeletedAt,
	}
}

func CourseFromModel(m CourseModel) catalog.Course {
	return catalog.Course{
		ID:        m.ID,
		Name:      m.Name,
		Code:      m.Code,
		CreatedAt: m.CreatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func GroupToModel(g catalog.Group) GroupModel {
	return GroupModel{
		ID:        g.ID,
		Name:      g.Name,
		CreatedAt: g.CreatedAt,
		DeletedAt: g.DeletedAt,
	}
}

func GroupFromModel(m GroupModel) catalog.Group {
	return catalog.Group{
		ID:        m.ID,
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func StreamToModel(s catalog.Stream) StreamModel {
	return StreamModel{
		ID:        s.ID,
		CourseID:  s.CourseID,
		Name:      s.Name,
		CreatedAt: s.CreatedAt,
		DeletedAt: s.DeletedAt,
	}
}

func StreamFromModel(m StreamModel, groupIDs []uuid.UUID) catalog.Stream {
	return catalog.Stream{
		ID:        m.ID,
		CourseID:  m.CourseID,
		Name:      m.Name,
		GroupIDs:  groupIDs,
		CreatedAt: m.CreatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func ClassroomToModel(c catalog.Classroom) (ClassroomModel, error) {
	bssids, err := MarshalToJSONB(c.AllowedBSSIDs)
	if err != nil {
		return ClassroomModel{}, err
	}
	return ClassroomModel{
		ID:            c.ID,
		Building:      c.Building,
		RoomNumber:    c.RoomNumber,
		Latitude:      c.Latitude,
		Longitude:     c.Longitude,
		RadiusM:       c.RadiusMeters,
		AllowedBSSIDs: bssids,
		CreatedAt:     c.CreatedAt,
		DeletedAt:     c.DeletedAt,
	}, nil
}

func ClassroomFromModel(m ClassroomModel) (catalog.Classroom, error) {
	var bssids []string
	if err := UnmarshalFromJSONB(m.AllowedBSSIDs, &bssids); err != nil {
		return catalog.Classroom{}, err
	}
	return catalog.Classroom{
		ID:            m.ID,
		Building:      m.Building,
		RoomNumber:    m.RoomNumber,
		Latitude:      m.Latitude,
		Longitude:     m.Longitude,
		RadiusMeters:  m.RadiusM,
		AllowedBSSIDs: bssids,
		CreatedAt:     m.CreatedAt,
		DeletedAt:     m.DeletedAt,
	}, nil
}

// ============================================================================
// Policy mappers
// ============================================================================

func PolicyToModel(p policy.SecurityPolicy) (SecurityPolicyModel, error) {
	mech, err := MarshalToJSONB(p.Mechanisms)
	if err != nil {
		return SecurityPolicyModel{}, err
	}
	return SecurityPolicyModel{
		ID:         p.ID,
		Name:       p.Name,
		Mechanisms: mech,
		IsDefault:  p.IsDefault,
		CreatedBy:  p.CreatedBy,
		CreatedAt:  p.CreatedAt,
		DeletedAt:  p.DeletedAt,
	}, nil
}

func PolicyFromModel(m SecurityPolicyModel) (policy.SecurityPolicy, error) {
	var cfg policy.MechanismsConfig
	if err := UnmarshalFromJSONB(m.Mechanisms, &cfg); err != nil {
		return policy.SecurityPolicy{}, err
	}
	return policy.SecurityPolicy{
		ID:         m.ID,
		Name:       m.Name,
		Mechanisms: cfg,
		IsDefault:  m.IsDefault,
		CreatedBy:  m.CreatedBy,
		CreatedAt:  m.CreatedAt,
		DeletedAt:  m.DeletedAt,
	}, nil
}

// ============================================================================
// Session mappers (M:N groups хранится в session_groups, загружается отдельно)
// ============================================================================

func SessionToModel(s session.Session) SessionModel {
	return SessionModel{
		ID:               s.ID,
		TeacherID:        s.TeacherID,
		CourseID:         s.CourseID,
		ClassroomID:      s.ClassroomID,
		SecurityPolicyID: s.SecurityPolicyID,
		StartsAt:         s.StartsAt,
		EndsAt:           s.EndsAt,
		Status:           string(s.Status),
		QRSecret:         s.QRSecret,
		QRTTLSeconds:     s.QRTTLSeconds,
		QRCounter:        s.QRCounter,
		CreatedAt:        s.CreatedAt,
	}
}

func SessionFromModel(m SessionModel, groupIDs []uuid.UUID) session.Session {
	return session.Session{
		ID:               m.ID,
		TeacherID:        m.TeacherID,
		CourseID:         m.CourseID,
		ClassroomID:      m.ClassroomID,
		SecurityPolicyID: m.SecurityPolicyID,
		StartsAt:         m.StartsAt,
		EndsAt:           m.EndsAt,
		Status:           session.Status(m.Status),
		QRSecret:         m.QRSecret,
		QRTTLSeconds:     m.QRTTLSeconds,
		QRCounter:        m.QRCounter,
		GroupIDs:         groupIDs,
		CreatedAt:        m.CreatedAt,
	}
}

// ============================================================================
// Attendance mappers
// ============================================================================

func AttendanceRecordToModel(r attendance.Record) AttendanceRecordModel {
	var finalStatus *string
	if r.FinalStatus != nil {
		s := string(*r.FinalStatus)
		finalStatus = &s
	}
	return AttendanceRecordModel{
		ID:                r.ID,
		SessionID:         r.SessionID,
		StudentID:         r.StudentID,
		SubmittedAt:       r.SubmittedAt,
		SubmittedQRToken:  r.SubmittedQRToken,
		PreliminaryStatus: string(r.PreliminaryStatus),
		FinalStatus:       finalStatus,
		ResolvedBy:        r.ResolvedBy,
		ResolvedAt:        r.ResolvedAt,
		Notes:             r.Notes,
	}
}

func AttendanceRecordFromModel(m AttendanceRecordModel) attendance.Record {
	var finalStatus *attendance.Status
	if m.FinalStatus != nil {
		s := attendance.Status(*m.FinalStatus)
		finalStatus = &s
	}
	return attendance.Record{
		ID:                m.ID,
		SessionID:         m.SessionID,
		StudentID:         m.StudentID,
		SubmittedAt:       m.SubmittedAt,
		SubmittedQRToken:  m.SubmittedQRToken,
		PreliminaryStatus: attendance.Status(m.PreliminaryStatus),
		FinalStatus:       finalStatus,
		ResolvedBy:        m.ResolvedBy,
		ResolvedAt:        m.ResolvedAt,
		Notes:             m.Notes,
	}
}

func CheckResultToModel(c attendance.CheckResult) (SecurityCheckResultModel, error) {
	details, err := MarshalToJSONB(c.Details)
	if err != nil {
		return SecurityCheckResultModel{}, err
	}
	return SecurityCheckResultModel{
		ID:           c.ID,
		AttendanceID: c.AttendanceID,
		Mechanism:    c.Mechanism,
		Status:       string(c.Status),
		Details:      details,
		CheckedAt:    c.CheckedAt,
	}, nil
}

func CheckResultFromModel(m SecurityCheckResultModel) (attendance.CheckResult, error) {
	var details map[string]any
	if err := UnmarshalFromJSONB(m.Details, &details); err != nil {
		return attendance.CheckResult{}, err
	}
	return attendance.CheckResult{
		ID:           m.ID,
		AttendanceID: m.AttendanceID,
		Mechanism:    m.Mechanism,
		Status:       attendance.CheckStatus(m.Status),
		Details:      details,
		CheckedAt:    m.CheckedAt,
	}, nil
}

// ============================================================================
// Audit mappers
// ============================================================================

func AuditEntryToModel(e audit.Entry) (AuditLogModel, error) {
	payload, err := MarshalToJSONB(e.Payload)
	if err != nil {
		return AuditLogModel{}, err
	}
	var ip *string
	if e.IPAddress != nil {
		s := e.IPAddress.String()
		ip = &s
	}
	return AuditLogModel{
		ID:         e.ID,
		PrevHash:   e.PrevHash,
		RecordHash: e.RecordHash,
		OccurredAt: e.OccurredAt,
		ActorID:    e.ActorID,
		ActorRole:  e.ActorRole,
		Action:     string(e.Action),
		EntityType: e.EntityType,
		EntityID:   e.EntityID,
		Payload:    payload,
		IPAddress:  ip,
		UserAgent:  e.UserAgent,
	}, nil
}

func AuditEntryFromModel(m AuditLogModel) (audit.Entry, error) {
	var payload map[string]any
	if err := UnmarshalFromJSONB(m.Payload, &payload); err != nil {
		return audit.Entry{}, err
	}
	var ip net.IP
	if m.IPAddress != nil && *m.IPAddress != "" {
		ip = net.ParseIP(stripCIDR(*m.IPAddress))
	}
	return audit.Entry{
		ID:         m.ID,
		PrevHash:   m.PrevHash,
		RecordHash: m.RecordHash,
		OccurredAt: m.OccurredAt,
		ActorID:    m.ActorID,
		ActorRole:  m.ActorRole,
		Action:     audit.Action(m.Action),
		EntityType: m.EntityType,
		EntityID:   m.EntityID,
		Payload:    payload,
		IPAddress:  ip,
		UserAgent:  m.UserAgent,
	}, nil
}

// stripCIDR убирает /NN из inet, если Postgres отдал с маской (inet = ip[/cidr]).
// Для скалярных адресов pgx обычно отдаёт без маски, но на всякий случай.
func stripCIDR(s string) string {
	for i, c := range s {
		if c == '/' {
			return s[:i]
		}
	}
	return s
}
