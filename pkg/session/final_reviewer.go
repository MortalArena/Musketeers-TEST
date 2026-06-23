package session

import (
	"time"
)

// FinalReviewer يراجع المشروع نهائياً
type FinalReviewer struct{}

// ReviewResult نتيجة المراجعة
type ReviewResult struct {
	Passed      bool          `json:"passed"`
	Score       float64       `json:"score"` // 0-100
	Issues      []ReviewIssue `json:"issues"`
	Suggestions []string      `json:"suggestions"`
	ReviewedAt  time.Time     `json:"reviewed_at"`
	Duration    time.Duration `json:"duration"`
}

// ReviewIssue مشكلة في المراجعة
type ReviewIssue struct {
	Severity    string `json:"severity"` // critical, high, medium, low
	Category    string `json:"category"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Suggestion  string `json:"suggestion"`
}

// NewFinalReviewer ينشئ مراجع نهائي
func NewFinalReviewer() *FinalReviewer {
	return &FinalReviewer{}
}

// Review يراجع المشروع نهائياً
func (fr *FinalReviewer) Review(final *FinalArtifact) (*ReviewResult, error) {
	startTime := time.Now()

	result := &ReviewResult{
		Passed:      true,
		Score:       100.0,
		Issues:      make([]ReviewIssue, 0),
		Suggestions: make([]string, 0),
		ReviewedAt:  startTime,
	}

	// 1. مراجعة استيفاء المتطلبات
	reqIssues := fr.checkRequirements(final)
	result.Issues = append(result.Issues, reqIssues...)

	// 2. مراجعة التكامل
	integrationIssues := fr.checkIntegration(final)
	result.Issues = append(result.Issues, integrationIssues...)

	// 3. مراجعة الأمان
	securityIssues := fr.checkSecurity(final)
	result.Issues = append(result.Issues, securityIssues...)

	// 4. مراجعة الأداء
	performanceIssues := fr.checkPerformance(final)
	result.Issues = append(result.Issues, performanceIssues...)

	// 5. مراجعة التوثيق
	docIssues := fr.checkDocumentation(final)
	result.Issues = append(result.Issues, docIssues...)

	// حساب النتيجة
	criticalCount := 0
	for _, issue := range result.Issues {
		switch issue.Severity {
		case "critical":
			criticalCount++
			result.Score -= 20
		case "high":
			result.Score -= 10
		case "medium":
			result.Score -= 5
		case "low":
			result.Score -= 2
		}
	}

	if result.Score < 0 {
		result.Score = 0
	}

	result.Passed = result.Score >= 70.0 && criticalCount == 0
	result.Duration = time.Since(startTime)

	return result, nil
}

func (fr *FinalReviewer) checkRequirements(final *FinalArtifact) []ReviewIssue {
	issues := []ReviewIssue{}

	// [SAFETY] التحقق من وجود القطع الأثرية
	if len(final.Artifacts) == 0 {
		issues = append(issues, ReviewIssue{
			Severity:    "critical",
			Category:    "requirements",
			Description: "No artifacts found in the project",
			Location:    "root",
			Suggestion:  "Ensure at least one artifact is created",
		})
	}

	// [SAFETY] التحقق من وجود اسم المشروع
	if final.Name == "" {
		issues = append(issues, ReviewIssue{
			Severity:    "high",
			Category:    "requirements",
			Description: "Project name is empty",
			Location:    "metadata",
			Suggestion:  "Set a meaningful project name",
		})
	}

	// [SAFETY] التحقق من وجود وصف
	if final.Description == "" {
		issues = append(issues, ReviewIssue{
			Severity:    "medium",
			Category:    "requirements",
			Description: "Project description is missing",
			Location:    "metadata",
			Suggestion:  "Add a project description",
		})
	}

	// [SAFETY] التحقق من الحجم الكلي
	if final.TotalSize == 0 {
		issues = append(issues, ReviewIssue{
			Severity:    "high",
			Category:    "requirements",
			Description: "Project total size is zero",
			Location:    "metadata",
			Suggestion:  "Ensure artifacts contain data",
		})
	}

	return issues
}

func (fr *FinalReviewer) checkIntegration(final *FinalArtifact) []ReviewIssue {
	issues := []ReviewIssue{}

	// [SAFETY] التحقق من تكامل القطع الأثرية
	for _, artifact := range final.Artifacts {
		if artifact.ID == "" {
			issues = append(issues, ReviewIssue{
				Severity:    "high",
				Category:    "integration",
				Description: "Artifact ID is empty",
				Location:    artifact.Name,
				Suggestion:  "Ensure all artifacts have unique IDs",
			})
		}
		if artifact.Name == "" {
			issues = append(issues, ReviewIssue{
				Severity:    "high",
				Category:    "integration",
				Description: "Artifact name is empty",
				Location:    artifact.ID,
				Suggestion:  "Set a meaningful name for the artifact",
			})
		}
		if artifact.Type == "" {
			issues = append(issues, ReviewIssue{
				Severity:    "medium",
				Category:    "integration",
				Description: "Artifact type is not specified",
				Location:    artifact.ID,
				Suggestion:  "Specify the artifact type (file, data, url)",
			})
		}
	}

	// [SAFETY] التحقق من هيكل الملفات
	if len(final.Structure) == 0 {
		issues = append(issues, ReviewIssue{
			Severity:    "medium",
			Category:    "integration",
			Description: "Project structure is empty",
			Location:    "structure",
			Suggestion:  "Build a proper file structure",
		})
	}

	return issues
}

func (fr *FinalReviewer) checkSecurity(final *FinalArtifact) []ReviewIssue {
	issues := []ReviewIssue{}

	// [SAFETY] التحقق من سلامة البيانات
	for _, artifact := range final.Artifacts {
		if artifact.Checksum == "" {
			issues = append(issues, ReviewIssue{
				Severity:    "high",
				Category:    "security",
				Description: "Artifact checksum is missing",
				Location:    artifact.ID,
				Suggestion:  "Calculate and store checksum for integrity verification",
			})
		}
		if artifact.Size > MaxArtifactSize {
			issues = append(issues, ReviewIssue{
				Severity:    "medium",
				Category:    "security",
				Description: "Artifact size exceeds recommended limit",
				Location:    artifact.ID,
				Suggestion:  "Consider splitting large artifacts",
			})
		}
	}

	// [SAFETY] التحقق من الحجم الكلي
	if final.TotalSize > MaxTotalSize {
		issues = append(issues, ReviewIssue{
			Severity:    "medium",
			Category:    "security",
			Description: "Project total size exceeds recommended limit",
			Location:    "metadata",
			Suggestion:  "Consider optimizing or compressing the project",
		})
	}

	return issues
}

func (fr *FinalReviewer) checkPerformance(final *FinalArtifact) []ReviewIssue {
	issues := []ReviewIssue{}

	// [SAFETY] التحقق من عدد القطع الأثرية
	if len(final.Artifacts) > MaxArtifacts {
		issues = append(issues, ReviewIssue{
			Severity:    "medium",
			Category:    "performance",
			Description: "Too many artifacts may impact performance",
			Location:    "metadata",
			Suggestion:  "Consider consolidating or organizing artifacts",
		})
	}

	// [SAFETY] التحقق من حجم القطع الأثرية الكبير
	for _, artifact := range final.Artifacts {
		if artifact.Size > 10*1024*1024 { // 10MB
			issues = append(issues, ReviewIssue{
				Severity:    "low",
				Category:    "performance",
				Description: "Large artifact may impact performance",
				Location:    artifact.ID,
				Suggestion:  "Consider compression or lazy loading",
			})
		}
	}

	return issues
}

func (fr *FinalReviewer) checkDocumentation(final *FinalArtifact) []ReviewIssue {
	issues := []ReviewIssue{}

	// [SAFETY] التحقق من وجود وصف المشروع
	if final.Description == "" {
		issues = append(issues, ReviewIssue{
			Severity:    "medium",
			Category:    "documentation",
			Description: "Project description is missing",
			Location:    "metadata",
			Suggestion:  "Add a comprehensive project description",
		})
	}

	// [SAFETY] التحقق من وجود README
	hasReadme := false
	for _, artifact := range final.Artifacts {
		if artifact.Name == "README.md" || artifact.Name == "README" {
			hasReadme = true
			break
		}
	}

	if !hasReadme {
		issues = append(issues, ReviewIssue{
			Severity:    "low",
			Category:    "documentation",
			Description: "README file is missing",
			Location:    "root",
			Suggestion:  "Add a README.md file with project documentation",
		})
	}

	// [SAFETY] التحقق من وصف القطع الأثرية
	for _, artifact := range final.Artifacts {
		if artifact.Metadata == nil || len(artifact.Metadata) == 0 {
			issues = append(issues, ReviewIssue{
				Severity:    "low",
				Category:    "documentation",
				Description: "Artifact metadata is missing",
				Location:    artifact.ID,
				Suggestion:  "Add metadata to describe the artifact purpose",
			})
		}
	}

	return issues
}
