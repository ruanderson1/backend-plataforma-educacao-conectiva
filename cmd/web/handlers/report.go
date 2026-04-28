package handlers

import (
	"plataforma/internal/classroom"
)

type ReportHandler struct {
	Handler *classroom.ReportHandler
}

func NewReportHandler(handler *classroom.ReportHandler) *ReportHandler {
	return &ReportHandler{Handler: handler}
}
