// Package paperscan provides the PaperScan extraction layer.
//
// ARCHITECTURE OVERVIEW
// ─────────────────────
// PaperScan is an INPUT AUTOMATION layer that sits BEFORE the existing
// MarkAI calculation engine. Its only job is to attempt to read individual
// question marks from an image of the evaluated marks section and return
// them with a confidence status so the faculty can verify before submitting.
//
// PaperScan NEVER calculates totals or percentages.
// The existing calculator package remains the single source of truth.
//
// EXTRACTION ENGINE STATUS
// ────────────────────────
// Tesseract OCR is not installed on this machine and no handwriting
// recognition library is available. Per the spec (§12), the correct
// approach is a CLEAN, PLUGGABLE INTERFACE — not faked results.
//
// The Extractor interface below is the integration point. When a real OCR
// engine (e.g. Tesseract, EasyOCR via subprocess, or a local model) becomes
// available, implement the Extractor interface and swap it into NewExtractor().
//
// The DemoExtractor is a transparent stub: it returns every field as
// "Needs Verification" with an empty value, which forces the faculty to
// fill in all marks manually. This is honest and safe.
package paperscan

import "mime/multipart"

// ─────────────────────────────────────────────
// CONFIDENCE STATUSES
// ─────────────────────────────────────────────

// Status describes how confident the extraction engine is about a value.
type Status string

const (
	// StatusExtracted means the engine read the value with high confidence.
	StatusExtracted Status = "Extracted"

	// StatusNeedsVerification means a value was detected but is uncertain.
	// The faculty MUST confirm before this value can be used.
	StatusNeedsVerification Status = "Needs Verification"

	// StatusNotDetected means no value could be read for this field.
	// The faculty MUST enter the value manually.
	StatusNotDetected Status = "Not Detected"
)

// ─────────────────────────────────────────────
// FIELD RESULT
// ─────────────────────────────────────────────

// MarkField holds an extracted numerical mark and its confidence status.
type MarkField struct {
	Value  int    // extracted value (0 if not detected)
	Status Status // confidence level
	Note   string // optional human-readable note (e.g. "multiple candidates")
}

// OptionField holds an extracted A/B option selection and its confidence status.
type OptionField struct {
	Value  string // "A", "B", or "" if not detected
	Status Status
	Note   string
}

// ─────────────────────────────────────────────
// EXTRACTION RESULT
// ─────────────────────────────────────────────

// ExtractionResult is the structured output from the extraction layer.
// Every field maps directly to an input in the existing MarkAI Mark Entry form.
// All fields are always present — the Status indicates what the engine found.
//
// This struct is passed to the faculty verification screen.
// The faculty can edit any field before confirming.
//
// IMPORTANT: This struct is NEVER used to calculate a total.
// Values flow from here into the existing /calculate handler.
type ExtractionResult struct {
	// ── Part A (Q1–Q5, max 2 each) ───────────────────────────
	Q1 MarkField
	Q2 MarkField
	Q3 MarkField
	Q4 MarkField
	Q5 MarkField

	// ── Part B (2 questions, A or B option, max 16 each) ─────
	B1Option OptionField
	B1Mark   MarkField

	B2Option OptionField
	B2Mark   MarkField

	// ── Part C (1 question, A or B option, max 8) ────────────
	C3Option OptionField
	C3Mark   MarkField

	// ── Optional student identity ────────────────────────────
	// Identity extraction is optional in v1. If the engine detects
	// a name or register number it populates these fields.
	Name  MarkField // reuses MarkField.Note for string value
	RegNo MarkField

	// ── Engine metadata ──────────────────────────────────────
	EngineUsed string // e.g. "demo", "tesseract", "easyocr"
	EngineNote string // human-readable note about engine limitations
}

// ─────────────────────────────────────────────
// EXTRACTOR INTERFACE
// ─────────────────────────────────────────────

// Extractor is the integration point for any OCR/extraction engine.
// To add a real OCR engine:
//  1. Create a new struct (e.g. TesseractExtractor) in this package.
//  2. Implement the Extract method.
//  3. Return it from NewExtractor() when the engine is available.
//
// The image is provided as a multipart.File so the handler does not
// need to save it to disk before passing it here (though the
// implementation may choose to do so for preprocessing).
type Extractor interface {
	Extract(file multipart.File, filename string) (ExtractionResult, error)
}

// ─────────────────────────────────────────────
// DEMO EXTRACTOR
// ─────────────────────────────────────────────

// DemoExtractor is a transparent stub used when no real OCR engine is
// available. It returns every field as StatusNotDetected so the faculty
// must enter all marks manually.
//
// This is intentionally honest: it does NOT guess, invent, or silently
// return plausible-looking values.
//
// When a real OCR engine is integrated, DemoExtractor can be kept as
// a fallback or removed.
type DemoExtractor struct{}

// Extract implements Extractor for DemoExtractor.
// All fields are returned as NotDetected.
func (d DemoExtractor) Extract(file multipart.File, filename string) (ExtractionResult, error) {
	notDetected := func(note string) MarkField {
		return MarkField{Value: 0, Status: StatusNotDetected, Note: note}
	}
	optNotDetected := func() OptionField {
		return OptionField{Value: "", Status: StatusNotDetected}
	}

	return ExtractionResult{
		Q1:       notDetected(""),
		Q2:       notDetected(""),
		Q3:       notDetected(""),
		Q4:       notDetected(""),
		Q5:       notDetected(""),
		B1Option: optNotDetected(),
		B1Mark:   notDetected(""),
		B2Option: optNotDetected(),
		B2Mark:   notDetected(""),
		C3Option: optNotDetected(),
		C3Mark:   notDetected(""),
		Name:     notDetected(""),
		RegNo:    notDetected(""),

		EngineUsed: "demo",
		EngineNote: "No OCR engine is available on this server. " +
			"Tesseract is not installed. " +
			"All fields require manual entry. " +
			"To enable automatic extraction, install Tesseract and implement TesseractExtractor.",
	}, nil
}

// ─────────────────────────────────────────────
// FACTORY
// ─────────────────────────────────────────────

// NewExtractor returns the best available extraction engine.
//
// Priority order (first available wins):
//  1. TesseractExtractor  — when Tesseract is installed       [NOT YET IMPLEMENTED]
//  2. EasyOCRExtractor    — when EasyOCR subprocess is ready  [NOT YET IMPLEMENTED]
//  3. DemoExtractor       — transparent stub, always available
//
// To plug in a real engine: implement the Extractor interface and add a
// detection check (e.g. exec.LookPath("tesseract")) here.
func NewExtractor() Extractor {
	// Future: check for tesseract, easyocr, etc.
	// if _, err := exec.LookPath("tesseract"); err == nil {
	//     return TesseractExtractor{}
	// }
	return DemoExtractor{}
}

// ─────────────────────────────────────────────
// HELPERS
// ─────────────────────────────────────────────

// IsFullyExtracted returns true only when every mandatory field was
// extracted with StatusExtracted (high confidence).
// Used to decide whether to show a prominent "please verify" warning.
func IsFullyExtracted(r ExtractionResult) bool {
	fields := []Status{
		r.Q1.Status, r.Q2.Status, r.Q3.Status, r.Q4.Status, r.Q5.Status,
		r.B1Mark.Status, r.B1Option.Status,
		r.B2Mark.Status, r.B2Option.Status,
		r.C3Mark.Status, r.C3Option.Status,
	}
	for _, s := range fields {
		if s != StatusExtracted {
			return false
		}
	}
	return true
}

// StatusBadgeClass maps a Status to the CSS class used in the UI.
func StatusBadgeClass(s Status) string {
	switch s {
	case StatusExtracted:
		return "ps-badge-extracted"
	case StatusNeedsVerification:
		return "ps-badge-verify"
	default:
		return "ps-badge-missing"
	}
}
