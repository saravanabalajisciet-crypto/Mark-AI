package models

// Student holds the marks and calculated results
// for a single internal assessment submission.
type Student struct {
	Name       string
	RegNo      string
	PartATotal int
	PartBTotal int
	PartCTotal int
	Total      int
	Percentage float64
}

// ClassStats holds aggregate statistics computed
// from all students stored in the class dataset.
// Every field is derived — nothing is hardcoded.
type ClassStats struct {
	TotalStudents  int
	Passed         int
	NeedsReview    int
	ClassAverage   float64
	HighestTotal   int
	LowestTotal    int
	PassPercentage float64
	PassMark       int // the threshold used for this calculation
}
