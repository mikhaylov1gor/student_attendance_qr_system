package dto

import (
	"time"

	"github.com/google/uuid"

	"attendance/internal/domain/catalog"
)

// =========================================================================
// Course
// =========================================================================

type CourseResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	CreatedAt time.Time `json:"created_at"`
}

func CourseFromDomain(c catalog.Course) CourseResponse {
	return CourseResponse{ID: c.ID.String(), Name: c.Name, Code: c.Code, CreatedAt: c.CreatedAt}
}

type CreateCourseRequest struct {
	Name string `json:"name" validate:"required,min=1,max=128"`
	Code string `json:"code" validate:"required,min=1,max=32"`
}

type UpdateCourseRequest struct {
	Name *string `json:"name,omitempty" validate:"omitempty,min=1,max=128"`
	Code *string `json:"code,omitempty" validate:"omitempty,min=1,max=32"`
}

type CourseListResponse struct {
	Items []CourseResponse `json:"items"`
	Total int              `json:"total"`
}

// =========================================================================
// Group
// =========================================================================

type GroupResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func GroupFromDomain(g catalog.Group) GroupResponse {
	return GroupResponse{ID: g.ID.String(), Name: g.Name, CreatedAt: g.CreatedAt}
}

type CreateGroupRequest struct {
	Name string `json:"name" validate:"required,min=1,max=64"`
}

type UpdateGroupRequest struct {
	Name *string `json:"name,omitempty" validate:"omitempty,min=1,max=64"`
}

type GroupListResponse struct {
	Items []GroupResponse `json:"items"`
	Total int             `json:"total"`
}

// =========================================================================
// Stream
// =========================================================================

type StreamResponse struct {
	ID        string    `json:"id"`
	CourseID  string    `json:"course_id"`
	Name      string    `json:"name"`
	GroupIDs  []string  `json:"group_ids"`
	CreatedAt time.Time `json:"created_at"`
}

func StreamFromDomain(s catalog.Stream) StreamResponse {
	gids := make([]string, len(s.GroupIDs))
	for i, g := range s.GroupIDs {
		gids[i] = g.String()
	}
	return StreamResponse{
		ID: s.ID.String(), CourseID: s.CourseID.String(),
		Name: s.Name, GroupIDs: gids, CreatedAt: s.CreatedAt,
	}
}

type CreateStreamRequest struct {
	CourseID uuid.UUID   `json:"course_id" validate:"required"`
	Name     string      `json:"name"      validate:"required,min=1,max=128"`
	GroupIDs []uuid.UUID `json:"group_ids" validate:"required,min=1,dive,required"`
}

type UpdateStreamRequest struct {
	Name     *string      `json:"name,omitempty"      validate:"omitempty,min=1,max=128"`
	GroupIDs *[]uuid.UUID `json:"group_ids,omitempty" validate:"omitempty,min=1,dive,required"`
}

type StreamListResponse struct {
	Items []StreamResponse `json:"items"`
	Total int              `json:"total"`
}

// =========================================================================
// Classroom
// =========================================================================

type ClassroomResponse struct {
	ID            string    `json:"id"`
	Building      string    `json:"building"`
	RoomNumber    string    `json:"room_number"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	RadiusMeters  int       `json:"radius_m"`
	AllowedBSSIDs []string  `json:"allowed_bssids"`
	CreatedAt     time.Time `json:"created_at"`
}

func ClassroomFromDomain(c catalog.Classroom) ClassroomResponse {
	return ClassroomResponse{
		ID:            c.ID.String(),
		Building:      c.Building,
		RoomNumber:    c.RoomNumber,
		Latitude:      c.Latitude,
		Longitude:     c.Longitude,
		RadiusMeters:  c.RadiusMeters,
		AllowedBSSIDs: c.AllowedBSSIDs,
		CreatedAt:     c.CreatedAt,
	}
}

type CreateClassroomRequest struct {
	Building      string   `json:"building"       validate:"required,min=1,max=128"`
	RoomNumber    string   `json:"room_number"    validate:"required,min=1,max=32"`
	Latitude      float64  `json:"latitude"       validate:"required,min=-90,max=90"`
	Longitude     float64  `json:"longitude"      validate:"required,min=-180,max=180"`
	RadiusMeters  int      `json:"radius_m"       validate:"required,min=1,max=1000"`
	AllowedBSSIDs []string `json:"allowed_bssids" validate:"dive,max=64"`
}

type UpdateClassroomRequest struct {
	Building      *string   `json:"building,omitempty"       validate:"omitempty,min=1,max=128"`
	RoomNumber    *string   `json:"room_number,omitempty"    validate:"omitempty,min=1,max=32"`
	Latitude      *float64  `json:"latitude,omitempty"       validate:"omitempty,min=-90,max=90"`
	Longitude     *float64  `json:"longitude,omitempty"      validate:"omitempty,min=-180,max=180"`
	RadiusMeters  *int      `json:"radius_m,omitempty"       validate:"omitempty,min=1,max=1000"`
	AllowedBSSIDs *[]string `json:"allowed_bssids,omitempty" validate:"omitempty,dive,max=64"`
}

type ClassroomListResponse struct {
	Items []ClassroomResponse `json:"items"`
	Total int                 `json:"total"`
}
