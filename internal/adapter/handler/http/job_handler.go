package http

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/vantutran2k1/orbit/internal/core/service"
)

type JobHandler struct {
	svc *service.SchedulerService
}

func NewJobHandler(svc *service.SchedulerService) *JobHandler {
	return &JobHandler{svc: svc}
}

func (h *JobHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateJobRequest

	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid JSON"})
		return
	}

	if err := req.Validate(); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	// TODO: Use Auth to get tenant ID
	job := req.ToDomain("11111111-1111-1111-1111-111111111111")
	job.TenantID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

	if err := h.svc.ScheduleJob(r.Context(), job); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, JobResponse{
		ID:        job.ID.String(),
		Title:     job.Title,
		Status:    string(job.Status),
		NextRunAt: job.NextRunAt.String(),
	})
}
