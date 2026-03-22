package handlers

import (
	"plataforma/internal/classroom"
)

type ClassroomHandler struct {
	Handler *classroom.Handler
}

func NewClassroomHandler(handler *classroom.Handler) *ClassroomHandler {
	return &ClassroomHandler{Handler: handler}
}
