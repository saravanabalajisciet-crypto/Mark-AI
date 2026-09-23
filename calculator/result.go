package calculator

import "student-result-cli/models"

// ─────────────────────────────────────────────
// Per-student calculations
// ─────────────────────────────────────────────

// CalculatePartATotal adds up all 5 Part A marks.
// Each question is worth 2 marks, so the max is 10.
func CalculatePartATotal(q1, q2, q3, q4, q5 int) int {
	return q1 + q2 + q3 + q4 + q5
}

// CalculatePartBTotal adds the two chosen Part B marks.
// Each question allows Option A or B (max 16 each), so the max is 32.
func CalculatePartBTotal(b1Mark, b2Mark int) int {
	return b1Mark + b2Mark
}

// CalculatePartCTotal returns the chosen Part C mark.
// The single question allows Option A or B (max 8), so the max is 8.
func CalculatePartCTotal(c3Mark int) int {
	return c3Mark
}

// CalculateTotal sums Part A, Part B and Part C totals.
// The grand total is out of 50.
func CalculateTotal(partA, partB, partC int) int {
	return partA + partB + partC
}

// CalculatePercentage returns the percentage score out of 50.
// Example: total 38 → 76.00
func CalculatePercentage(total int) float64 {
	return (float64(total) / 50.0) * 100
}

// ─────────────────────────────────────────────
// Class-level calculations
// ─────────────────────────────────────────────

// ComputeClassStats derives aggregate statistics from a slice of students.
// passMark is the minimum total required to be considered "passed".
// Returns a zero-value ClassStats when the slice is empty.
func ComputeClassStats(students []models.Student, passMark int) models.ClassStats {
	n := len(students)
	if n == 0 {
		return models.ClassStats{PassMark: passMark}
	}

	passed := 0
	totalSum := 0
	highest := students[0].Total
	lowest := students[0].Total

	for _, s := range students {
		totalSum += s.Total

		if s.Total >= passMark {
			passed++
		}

		if s.Total > highest {
			highest = s.Total
		}
		if s.Total < lowest {
			lowest = s.Total
		}
	}

	needsReview := n - passed
	classAvg := float64(totalSum) / float64(n)
	passPct := (float64(passed) / float64(n)) * 100

	return models.ClassStats{
		TotalStudents:  n,
		Passed:         passed,
		NeedsReview:    needsReview,
		ClassAverage:   classAvg,
		HighestTotal:   highest,
		LowestTotal:    lowest,
		PassPercentage: passPct,
		PassMark:       passMark,
	}
}

// FilterStudents returns a subset of the students slice based on a named filter.
//
//	"below25"    → Total < 25
//	"above35"    → Total > 35
//	"between"    → 25 <= Total <= 35
//	""  or any   → all students (no filter)
func FilterStudents(students []models.Student, filter string) []models.Student {
	if filter == "" || filter == "all" {
		return students
	}

	result := make([]models.Student, 0)

	for _, s := range students {
		switch filter {
		case "below25":
			if s.Total < 25 {
				result = append(result, s)
			}
		case "above35":
			if s.Total > 35 {
				result = append(result, s)
			}
		case "between":
			if s.Total >= 25 && s.Total <= 35 {
				result = append(result, s)
			}
		}
	}

	return result
}
