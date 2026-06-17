package unified

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ProblemSolutionRegistry نظام تسجيل المشاكل والحلول
type ProblemSolutionRegistry struct {
	sessionID string
	logger    *zap.Logger
	problems  map[string]*Problem
	solutions map[string]*Solution
	mu        sync.RWMutex
}

// Problem مشكلة تم الإبلاغ عنها
type Problem struct {
	ID          string
	Description string
	Context     map[string]interface{}
	Timestamp   time.Time
	ReportedBy  string
	Status      string // "open", "solved"
}

// Solution حل لمشكلة
type Solution struct {
	ID          string
	ProblemID   string
	Description string
	Context     map[string]interface{}
	Timestamp   time.Time
	SolvedBy    string
	Verified    bool
}

// NewProblemSolutionRegistry ينشئ نظام تسجيل جديد
func NewProblemSolutionRegistry(sessionID string, logger *zap.Logger) *ProblemSolutionRegistry {
	return &ProblemSolutionRegistry{
		sessionID: sessionID,
		logger:    logger,
		problems:  make(map[string]*Problem),
		solutions: make(map[string]*Solution),
	}
}

// ReportProblem يبلغ عن مشكلة جديدة
func (psr *ProblemSolutionRegistry) ReportProblem(ctx context.Context, problem *Problem) error {
	psr.mu.Lock()
	defer psr.mu.Unlock()

	psr.problems[problem.ID] = problem

	psr.logger.Info("تم الإبلاغ عن مشكلة جديدة",
		zap.String("problem_id", problem.ID),
		zap.String("description", problem.Description),
		zap.String("reported_by", problem.ReportedBy),
	)

	return nil
}

// ReportSolution يبلغ عن حل لمشكلة
func (psr *ProblemSolutionRegistry) ReportSolution(ctx context.Context, solution *Solution) error {
	psr.mu.Lock()
	defer psr.mu.Unlock()

	psr.solutions[solution.ID] = solution

	// تحديث حالة المشكلة
	if problem, exists := psr.problems[solution.ProblemID]; exists {
		problem.Status = "solved"
	}

	psr.logger.Info("تم الإبلاغ عن حل جديد",
		zap.String("solution_id", solution.ID),
		zap.String("problem_id", solution.ProblemID),
		zap.String("description", solution.Description),
		zap.String("solved_by", solution.SolvedBy),
	)

	return nil
}

// SearchProblems يبحث عن مشاكل بناءً على الاستعلام
func (psr *ProblemSolutionRegistry) SearchProblems(query string) []*Problem {
	psr.mu.RLock()
	defer psr.mu.RUnlock()

	results := []*Problem{}

	for _, problem := range psr.problems {
		if strings.Contains(problem.Description, query) {
			results = append(results, problem)
		}
	}

	return results
}

// GetSolution يحصل على حل لمشكلة معينة
func (psr *ProblemSolutionRegistry) GetSolution(problemID string) *Solution {
	psr.mu.RLock()
	defer psr.mu.RUnlock()

	for _, solution := range psr.solutions {
		if solution.ProblemID == problemID {
			return solution
		}
	}

	return nil
}

// GetOpenProblems يحصل على جميع المشاكل المفتوحة
func (psr *ProblemSolutionRegistry) GetOpenProblems() []*Problem {
	psr.mu.RLock()
	defer psr.mu.RUnlock()

	results := []*Problem{}

	for _, problem := range psr.problems {
		if problem.Status == "open" {
			results = append(results, problem)
		}
	}

	return results
}

// GetSolvedProblems يحصل على جميع المشاكل المحلولة
func (psr *ProblemSolutionRegistry) GetSolvedProblems() []*Problem {
	psr.mu.RLock()
	defer psr.mu.RUnlock()

	results := []*Problem{}

	for _, problem := range psr.problems {
		if problem.Status == "solved" {
			results = append(results, problem)
		}
	}

	return results
}

// VerifySolution يتحقق من حل
func (psr *ProblemSolutionRegistry) VerifySolution(ctx context.Context, solutionID string) error {
	psr.mu.Lock()
	defer psr.mu.Unlock()

	if solution, exists := psr.solutions[solutionID]; exists {
		solution.Verified = true
		psr.logger.Info("تم التحقق من الحل",
			zap.String("solution_id", solutionID),
		)
		return nil
	}

	return fmt.Errorf("الحل غير موجود")
}
