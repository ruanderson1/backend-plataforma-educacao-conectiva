package handlers

import (
	"plataforma/internal/classroom"
)

type ClassroomHandler struct {
	Handler        *classroom.Handler
	StudentHandler *classroom.StudentHandler
}

func NewClassroomHandler(handler *classroom.Handler, studentHandler *classroom.StudentHandler) *ClassroomHandler {
	return &ClassroomHandler{Handler: handler, StudentHandler: studentHandler}
}
