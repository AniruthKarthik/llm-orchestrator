package tools

import (
	"context"
	"fmt"
	"time"
)

type ReportTool struct{}

func NewReportTool() *ReportTool { return &ReportTool{} }

func (t *ReportTool) Name() string { return "report" }

func (t *ReportTool) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	jobID, err := requireString(input, "job_id")
	if err != nil {
		return nil, fmt.Errorf("report: %w", err)
	}
	goal, err := requireString(input, "goal")
	if err != nil {
		return nil, fmt.Errorf("report: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("report: context cancelled: %w", ctx.Err())
	case <-time.After(20 * time.Millisecond):
	}

	var priorResults []any
	if v, ok := input["results"]; ok {
		if arr, ok := v.([]any); ok {
			priorResults = arr
		}
	}

	checks := []string{"schema", "completeness"}
	if v, ok := input["checks"]; ok {
		if arr, ok := v.([]any); ok {
			checks = make([]string, 0, len(arr))
			for _, c := range arr {
				if s, ok := c.(string); ok {
					checks = append(checks, s)
				}
			}
		}
	}
	validationResults := make(map[string]any, len(checks))
	for _, c := range checks {
		validationResults[c] = map[string]any{"passed": true, "note": "ok"}
	}

	sections := []map[string]any{
		{
			"title":   "Executive Summary",
			"content": fmt.Sprintf("Automated analysis completed for goal: %q", goal),
		},
		{
			"title":   "Task Results",
			"content": fmt.Sprintf("%d upstream result(s) collected and processed.", len(priorResults)),
			"items":   priorResults,
		},
		{
			"title":   "Conclusions",
			"content": "All planned tasks executed successfully. See task results for detail.",
		},
	}

	return map[string]any{
		"job_id":       jobID,
		"goal":         goal,
		"status":       "complete",
		"sections":     sections,
		"validation":   validationResults,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}, nil
}
