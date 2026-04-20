package catalog

import "errors"

var (
	ErrCourseNotFound    = errors.New("catalog: course not found")
	ErrCourseCodeTaken   = errors.New("catalog: course code already taken")
	ErrGroupNotFound     = errors.New("catalog: group not found")
	ErrGroupNameTaken    = errors.New("catalog: group name already taken")
	ErrStreamNotFound    = errors.New("catalog: stream not found")
	ErrClassroomNotFound = errors.New("catalog: classroom not found")
	// ErrInUse — попытка удалить справочную запись, на которую ссылаются другие сущности.
	ErrInUse = errors.New("catalog: entity is referenced by others")
)
