package handlers

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	appcatalog "attendance/internal/application/catalog"
	"attendance/internal/infrastructure/http/dto"
	"attendance/internal/infrastructure/http/httperr"
)

type CatalogHandler struct {
	svc *appcatalog.Service
	log *slog.Logger
}

func NewCatalogHandler(svc *appcatalog.Service, log *slog.Logger) *CatalogHandler {
	return &CatalogHandler{svc: svc, log: log}
}

// =========================================================================
// Courses
// =========================================================================

func (h *CatalogHandler) ListCourses(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListCourses(r.Context())
	if err != nil {
		httperr.LogUnexpected(h.log, r, err)
		httperr.Write(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	out := dto.CourseListResponse{Total: len(items), Items: make([]dto.CourseResponse, 0, len(items))}
	for _, c := range items {
		out.Items = append(out.Items, dto.CourseFromDomain(c))
	}
	httperr.WriteJSON(w, http.StatusOK, out)
}

func (h *CatalogHandler) GetCourse(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	c, err := h.svc.GetCourse(r.Context(), id)
	if err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	httperr.WriteJSON(w, http.StatusOK, dto.CourseFromDomain(c))
}

func (h *CatalogHandler) CreateCourse(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateCourseRequest
	if err := dto.Decode(w, r, &req); err != nil {
		return
	}
	c, err := h.svc.CreateCourse(r.Context(), appcatalog.CreateCourseInput{Name: req.Name, Code: req.Code})
	if err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	httperr.WriteJSON(w, http.StatusCreated, dto.CourseFromDomain(c))
}

func (h *CatalogHandler) UpdateCourse(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	var req dto.UpdateCourseRequest
	if err := dto.Decode(w, r, &req); err != nil {
		return
	}
	c, err := h.svc.UpdateCourse(r.Context(), id, appcatalog.UpdateCourseInput{Name: req.Name, Code: req.Code})
	if err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	httperr.WriteJSON(w, http.StatusOK, dto.CourseFromDomain(c))
}

func (h *CatalogHandler) DeleteCourse(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteCourse(r.Context(), id); err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// =========================================================================
// Groups
// =========================================================================

func (h *CatalogHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListGroups(r.Context())
	if err != nil {
		httperr.LogUnexpected(h.log, r, err)
		httperr.Write(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	out := dto.GroupListResponse{Total: len(items), Items: make([]dto.GroupResponse, 0, len(items))}
	for _, g := range items {
		out.Items = append(out.Items, dto.GroupFromDomain(g))
	}
	httperr.WriteJSON(w, http.StatusOK, out)
}

func (h *CatalogHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	g, err := h.svc.GetGroup(r.Context(), id)
	if err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	httperr.WriteJSON(w, http.StatusOK, dto.GroupFromDomain(g))
}

func (h *CatalogHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateGroupRequest
	if err := dto.Decode(w, r, &req); err != nil {
		return
	}
	g, err := h.svc.CreateGroup(r.Context(), appcatalog.CreateGroupInput{Name: req.Name})
	if err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	httperr.WriteJSON(w, http.StatusCreated, dto.GroupFromDomain(g))
}

func (h *CatalogHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	var req dto.UpdateGroupRequest
	if err := dto.Decode(w, r, &req); err != nil {
		return
	}
	g, err := h.svc.UpdateGroup(r.Context(), id, appcatalog.UpdateGroupInput{Name: req.Name})
	if err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	httperr.WriteJSON(w, http.StatusOK, dto.GroupFromDomain(g))
}

func (h *CatalogHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteGroup(r.Context(), id); err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// =========================================================================
// Streams
// =========================================================================

// ListStreams поддерживает обязательный query-параметр course_id.
func (h *CatalogHandler) ListStreams(w http.ResponseWriter, r *http.Request) {
	courseID, ok := parseUUIDQuery(w, r, "course_id")
	if !ok {
		return
	}
	items, err := h.svc.ListStreamsByCourse(r.Context(), courseID)
	if err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	out := dto.StreamListResponse{Total: len(items), Items: make([]dto.StreamResponse, 0, len(items))}
	for _, s := range items {
		out.Items = append(out.Items, dto.StreamFromDomain(s))
	}
	httperr.WriteJSON(w, http.StatusOK, out)
}

func (h *CatalogHandler) GetStream(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	s, err := h.svc.GetStream(r.Context(), id)
	if err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	httperr.WriteJSON(w, http.StatusOK, dto.StreamFromDomain(s))
}

func (h *CatalogHandler) CreateStream(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateStreamRequest
	if err := dto.Decode(w, r, &req); err != nil {
		return
	}
	s, err := h.svc.CreateStream(r.Context(), appcatalog.CreateStreamInput{
		CourseID: req.CourseID, Name: req.Name, GroupIDs: req.GroupIDs,
	})
	if err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	httperr.WriteJSON(w, http.StatusCreated, dto.StreamFromDomain(s))
}

func (h *CatalogHandler) UpdateStream(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	var req dto.UpdateStreamRequest
	if err := dto.Decode(w, r, &req); err != nil {
		return
	}
	s, err := h.svc.UpdateStream(r.Context(), id, appcatalog.UpdateStreamInput{
		Name: req.Name, GroupIDs: req.GroupIDs,
	})
	if err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	httperr.WriteJSON(w, http.StatusOK, dto.StreamFromDomain(s))
}

func (h *CatalogHandler) DeleteStream(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteStream(r.Context(), id); err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// =========================================================================
// Classrooms
// =========================================================================

func (h *CatalogHandler) ListClassrooms(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListClassrooms(r.Context())
	if err != nil {
		httperr.LogUnexpected(h.log, r, err)
		httperr.Write(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	out := dto.ClassroomListResponse{Total: len(items), Items: make([]dto.ClassroomResponse, 0, len(items))}
	for _, c := range items {
		out.Items = append(out.Items, dto.ClassroomFromDomain(c))
	}
	httperr.WriteJSON(w, http.StatusOK, out)
}

func (h *CatalogHandler) GetClassroom(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	c, err := h.svc.GetClassroom(r.Context(), id)
	if err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	httperr.WriteJSON(w, http.StatusOK, dto.ClassroomFromDomain(c))
}

func (h *CatalogHandler) CreateClassroom(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateClassroomRequest
	if err := dto.Decode(w, r, &req); err != nil {
		return
	}
	c, err := h.svc.CreateClassroom(r.Context(), appcatalog.CreateClassroomInput{
		Building:      req.Building,
		RoomNumber:    req.RoomNumber,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		RadiusMeters:  req.RadiusMeters,
		AllowedBSSIDs: req.AllowedBSSIDs,
	})
	if err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	httperr.WriteJSON(w, http.StatusCreated, dto.ClassroomFromDomain(c))
}

func (h *CatalogHandler) UpdateClassroom(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	var req dto.UpdateClassroomRequest
	if err := dto.Decode(w, r, &req); err != nil {
		return
	}
	c, err := h.svc.UpdateClassroom(r.Context(), id, appcatalog.UpdateClassroomInput{
		Building:      req.Building,
		RoomNumber:    req.RoomNumber,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		RadiusMeters:  req.RadiusMeters,
		AllowedBSSIDs: req.AllowedBSSIDs,
	})
	if err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	httperr.WriteJSON(w, http.StatusOK, dto.ClassroomFromDomain(c))
}

func (h *CatalogHandler) DeleteClassroom(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteClassroom(r.Context(), id); err != nil {
		httperr.RespondError(w, r, h.log, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// =========================================================================
// helpers
// =========================================================================

func parseUUIDQuery(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		httperr.Write(w, http.StatusBadRequest, "missing_"+name, "query param is required: "+name)
		return uuid.UUID{}, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		httperr.Write(w, http.StatusBadRequest, "invalid_"+name, "not a valid uuid")
		return uuid.UUID{}, false
	}
	return id, true
}
