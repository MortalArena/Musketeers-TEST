package templates

import (
	"fmt"

	"github.com/MortalArena/Musketeers/pkg/workflow"
)

const (
	CodeReviewWorkflowName   = "code-review"
	EmailSummaryWorkflowName = "email-summary"
	DailyReportWorkflowName  = "daily-report"
)

func Get(name string) (workflow.Workflow, error) {
	switch name {
	case CodeReviewWorkflowName:
		return codeReviewWorkflow(), nil
	case EmailSummaryWorkflowName:
		return emailSummaryWorkflow(), nil
	case DailyReportWorkflowName:
		return dailyReportWorkflow(), nil
	default:
		return workflow.Workflow{}, fmt.Errorf("workflow template not found: %s", name)
	}
}

func All() []workflow.Workflow {
	return []workflow.Workflow{codeReviewWorkflow(), emailSummaryWorkflow(), dailyReportWorkflow()}
}

func codeReviewWorkflow() workflow.Workflow {
	return workflow.Workflow{Name: CodeReviewWorkflowName, Description: "Review code changes", Steps: []workflow.Step{
		{Name: "fetch-pr", Type: workflow.StepCapability, Capability: "github.get_pr", Command: map[string]any{}},
		{Name: "review", Type: workflow.StepCapability, Capability: "review.code", Command: map[string]any{}},
	}}
}

func emailSummaryWorkflow() workflow.Workflow {
	return workflow.Workflow{Name: EmailSummaryWorkflowName, Description: "Summarize email inbox", Steps: []workflow.Step{
		{Name: "list-emails", Type: workflow.StepCapability, Capability: "gmail.list_emails", Command: map[string]any{}},
		{Name: "summarize", Type: workflow.StepCapability, Capability: "summarize.emails", Command: map[string]any{}},
	}}
}

func dailyReportWorkflow() workflow.Workflow {
	return workflow.Workflow{Name: DailyReportWorkflowName, Description: "Generate daily report", Steps: []workflow.Step{
		{Name: "collect", Type: workflow.StepCapability, Capability: "report.collect", Command: map[string]any{}},
		{Name: "send", Type: workflow.StepCapability, Capability: "gmail.send_email", Command: map[string]any{}},
	}}
}
