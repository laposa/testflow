package web

import (
	"net/http"
	"time"

	"github.com/laposa/testflow/app-go/internal/web/views"
)

// showExecutiveReport and showFullReport render the session reports (spec
// §8.11; legacy: SessionReportController::executive and ::full).
func (s *server) showExecutiveReport(w http.ResponseWriter, r *http.Request) {
	s.showReport(w, r, false)
}
func (s *server) showFullReport(w http.ResponseWriter, r *http.Request) { s.showReport(w, r, true) }

func (s *server) showReport(w http.ResponseWriter, r *http.Request, full bool) {
	id, ok := sessionID(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	ts, err := s.TestSessions.TestSession(ctx, id)
	if err != nil {
		s.pageError(w, r, "load test session", err)
		return
	}
	v := views.ReportView{Session: ts, Full: full, Now: time.Now()}
	if v.Runs, err = s.Runs.SessionRuns(ctx, id); err != nil {
		s.serverError(w, r, "load runs", err)
		return
	}
	if full {
		if v.Results, err = s.Runs.SessionResults(ctx, id); err != nil {
			s.serverError(w, r, "load results", err)
			return
		}
	}
	title := "Executive report"
	if full {
		title = "Full report"
	}
	s.render(w, r, http.StatusOK, views.ReportPage(s.page(r, title+" · "+ts.Name), v))
}
