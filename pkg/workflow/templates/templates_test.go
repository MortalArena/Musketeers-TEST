package templates

import "testing"

func TestBuiltInTemplates(t *testing.T) {
	templates := []string{CodeReviewWorkflowName, EmailSummaryWorkflowName, DailyReportWorkflowName}
	for _, name := range templates {
		wf, err := Get(name)
		if err != nil {
			t.Fatalf("Get(%s) returned error: %v", name, err)
		}
		if len(wf.Steps) == 0 {
			t.Fatalf("template %s has no steps", name)
		}
	}
}
