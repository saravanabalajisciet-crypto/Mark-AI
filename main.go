package main

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"student-result-cli/calculator"
	"student-result-cli/models"
	"student-result-cli/paperscan"
)

// ─────────────────────────────────────────────
// PASS MARK THRESHOLD
//
// Set to 25 for now. Change this once the college
// confirms the actual pass threshold for internal
// assessments out of 50.
// ─────────────────────────────────────────────
const PASS_MARK = 25

// ─────────────────────────────────────────────
// IN-MEMORY CLASS STORE
//
// A package-level slice that accumulates every
// successfully submitted student for the lifetime
// of the running server process.
//
// mu protects the slice from concurrent writes
// when multiple requests arrive at the same time.
// ─────────────────────────────────────────────
var (
	mu            sync.Mutex
	classStudents []models.Student
)

// addStudent safely appends a student to the store.
func addStudent(s models.Student) {
	mu.Lock()
	defer mu.Unlock()
	classStudents = append(classStudents, s)
}

// getStudents returns a safe copy of the current store.
func getStudents() []models.Student {
	mu.Lock()
	defer mu.Unlock()
	cp := make([]models.Student, len(classStudents))
	for i, s := range classStudents {
		cp[i] = s
	}
	return cp
}

// ─────────────────────────────────────────────
// TEMPLATES
// ─────────────────────────────────────────────

var tmplFuncs = template.FuncMap{
	"inc": func(i int) int { return i + 1 },
	// statusClass maps a paperscan.Status to its CSS badge class.
	"statusClass": paperscan.StatusBadgeClass,
}

var (
	tmplIndex = template.Must(
		template.New("index.html").Funcs(tmplFuncs).ParseFiles("templates/index.html"),
	)
	tmplClass = template.Must(
		template.New("class.html").Funcs(tmplFuncs).ParseFiles("templates/class.html"),
	)
	tmplPaperScan = template.Must(
		template.New("paperscan.html").Funcs(tmplFuncs).ParseFiles("templates/paperscan.html"),
	)
)

// ─────────────────────────────────────────────
// PAGE DATA TYPES
// ─────────────────────────────────────────────

// IndexPageData is passed to templates/index.html on every render.
type IndexPageData struct {
	Result   *models.Student
	ErrorMsg string
	Stats    models.ClassStats
}

// ClassPageData is passed to templates/class.html.
type ClassPageData struct {
	Students []models.Student
	Stats    models.ClassStats
	Filter   string
}

// PaperScanPageData is passed to templates/paperscan.html.
// Stage controls which section of the page is shown:
//
//	"upload"  — initial camera/upload UI (default)
//	"extract" — extraction result + verification form
//	"result"  — final calculated result (reuses IndexPageData patterns)
type PaperScanPageData struct {
	Stage      string                      // "upload" | "extract"
	Extraction *paperscan.ExtractionResult // nil on "upload" stage
	ErrorMsg   string
	Stats      models.ClassStats
}

// ─────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────
func main() {
	http.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(http.Dir("static"))),
	)

	// ── Existing routes (UNCHANGED) ──────────────────────────────
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/calculate", calculateHandler)
	http.HandleFunc("/class", classHandler)

	// ── PaperScan routes (NEW, additive) ─────────────────────────
	http.HandleFunc("/paperscan", paperScanHandler)
	http.HandleFunc("/paperscan/extract", paperScanExtractHandler)
	http.HandleFunc("/paperscan/confirm", paperScanConfirmHandler)

	fmt.Println("MarkAI server running at http://localhost:8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Server error:", err)
	}
}

// ─────────────────────────────────────────────
// EXISTING HANDLERS — UNCHANGED
// ─────────────────────────────────────────────

func homeHandler(w http.ResponseWriter, r *http.Request) {
	stats := calculator.ComputeClassStats(getStudents(), PASS_MARK)
	tmplIndex.Execute(w, IndexPageData{Stats: stats})
}

func calculateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	renderError := func(msg string) {
		stats := calculator.ComputeClassStats(getStudents(), PASS_MARK)
		tmplIndex.Execute(w, IndexPageData{ErrorMsg: msg, Stats: stats})
	}

	name := r.FormValue("name")
	regNo := r.FormValue("regno")

	if name == "" {
		renderError("Student name is required.")
		return
	}
	if regNo == "" {
		renderError("Register number is required.")
		return
	}

	q1, err := getIntMark(r, "q1", 2)
	if err != nil {
		renderError("Part A – Q1: " + err.Error())
		return
	}
	q2, err := getIntMark(r, "q2", 2)
	if err != nil {
		renderError("Part A – Q2: " + err.Error())
		return
	}
	q3, err := getIntMark(r, "q3", 2)
	if err != nil {
		renderError("Part A – Q3: " + err.Error())
		return
	}
	q4, err := getIntMark(r, "q4", 2)
	if err != nil {
		renderError("Part A – Q4: " + err.Error())
		return
	}
	q5, err := getIntMark(r, "q5", 2)
	if err != nil {
		renderError("Part A – Q5: " + err.Error())
		return
	}

	b1Mark, err := getOptionMark(r, "b1_option", "b1_mark", 16)
	if err != nil {
		renderError("Part B – Question 1: " + err.Error())
		return
	}
	b2Mark, err := getOptionMark(r, "b2_option", "b2_mark", 16)
	if err != nil {
		renderError("Part B – Question 2: " + err.Error())
		return
	}
	c3Mark, err := getOptionMark(r, "c3_option", "c3_mark", 8)
	if err != nil {
		renderError("Part C – Question 3: " + err.Error())
		return
	}

	partA := calculator.CalculatePartATotal(q1, q2, q3, q4, q5)
	partB := calculator.CalculatePartBTotal(b1Mark, b2Mark)
	partC := calculator.CalculatePartCTotal(c3Mark)
	total := calculator.CalculateTotal(partA, partB, partC)
	pct := calculator.CalculatePercentage(total)

	student := models.Student{
		Name:       name,
		RegNo:      regNo,
		PartATotal: partA,
		PartBTotal: partB,
		PartCTotal: partC,
		Total:      total,
		Percentage: pct,
	}

	addStudent(student)

	stats := calculator.ComputeClassStats(getStudents(), PASS_MARK)
	tmplIndex.Execute(w, IndexPageData{Result: &student, Stats: stats})
}

func classHandler(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("filter")
	all := getStudents()
	stats := calculator.ComputeClassStats(all, PASS_MARK)
	filtered := calculator.FilterStudents(all, filter)
	tmplClass.Execute(w, ClassPageData{
		Students: filtered,
		Stats:    stats,
		Filter:   filter,
	})
}

// ─────────────────────────────────────────────
// PAPERSCAN HANDLERS — NEW
// ─────────────────────────────────────────────

// paperScanHandler — GET /paperscan
// Renders the initial camera/upload page.
func paperScanHandler(w http.ResponseWriter, r *http.Request) {
	stats := calculator.ComputeClassStats(getStudents(), PASS_MARK)
	if err := tmplPaperScan.Execute(w, PaperScanPageData{
		Stage: "upload",
		Stats: stats,
	}); err != nil {
		fmt.Println("paperScanHandler template error:", err)
	}
}

// paperScanExtractHandler — POST /paperscan/extract
// Receives the uploaded/captured image, runs the extraction engine,
// and renders the verification form populated with the results.
func paperScanExtractHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/paperscan", http.StatusSeeOther)
		return
	}

	stats := calculator.ComputeClassStats(getStudents(), PASS_MARK)

	renderErr := func(msg string) {
		tmplPaperScan.Execute(w, PaperScanPageData{
			Stage:    "upload",
			ErrorMsg: msg,
			Stats:    stats,
		})
	}

	// Parse the multipart form (max 10 MB).
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		renderErr("Could not read the uploaded file. Please try again.")
		return
	}

	file, header, err := r.FormFile("marks_image")
	if err != nil {
		renderErr("Please upload a marks image.")
		return
	}
	defer file.Close()

	// Validate file type by extension.
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		renderErr("Unsupported file type. Please upload JPG, JPEG, or PNG.")
		return
	}

	// Run the extraction engine.
	extractor := paperscan.NewExtractor()
	result, err := extractor.Extract(file, header.Filename)
	if err != nil {
		renderErr("Could not process the image. Please try again with a clearer image.")
		return
	}

	// Render the verification form with the extraction result.
	if err := tmplPaperScan.Execute(w, PaperScanPageData{
		Stage:      "extract",
		Extraction: &result,
		Stats:      stats,
	}); err != nil {
		fmt.Println("paperScanExtractHandler template error:", err)
	}
}

// paperScanConfirmHandler — POST /paperscan/confirm
//
// Receives the faculty-verified mark values from the verification form
// and feeds them directly into the EXISTING calculation pipeline by
// forwarding an equivalent request to calculateHandler's shared logic.
//
// This handler does NOT duplicate any calculation or validation logic.
// It reuses getIntMark, getOptionMark, calculator.*, addStudent — the
// exact same functions used by calculateHandler.
func paperScanConfirmHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/paperscan", http.StatusSeeOther)
		return
	}

	stats := calculator.ComputeClassStats(getStudents(), PASS_MARK)

	// renderPSError re-renders the verify page with a validation message.
	// It rebuilds a minimal ExtractionResult from the posted form so the
	// faculty's edits are not lost on error.
	renderPSError := func(msg string) {
		result := rebuildExtractionFromForm(r)
		tmplPaperScan.Execute(w, PaperScanPageData{
			Stage:      "extract",
			Extraction: &result,
			ErrorMsg:   msg,
			Stats:      stats,
		})
	}

	// ── Student identity ─────────────────────────────────────────
	name := strings.TrimSpace(r.FormValue("name"))
	regNo := strings.TrimSpace(r.FormValue("regno"))

	if name == "" {
		renderPSError("Student name is required.")
		return
	}
	if regNo == "" {
		renderPSError("Register number is required.")
		return
	}

	// ── Part A (reuse existing getIntMark) ───────────────────────
	q1, err := getIntMark(r, "q1", 2)
	if err != nil {
		renderPSError("Part A – Q1: " + err.Error())
		return
	}
	q2, err := getIntMark(r, "q2", 2)
	if err != nil {
		renderPSError("Part A – Q2: " + err.Error())
		return
	}
	q3, err := getIntMark(r, "q3", 2)
	if err != nil {
		renderPSError("Part A – Q3: " + err.Error())
		return
	}
	q4, err := getIntMark(r, "q4", 2)
	if err != nil {
		renderPSError("Part A – Q4: " + err.Error())
		return
	}
	q5, err := getIntMark(r, "q5", 2)
	if err != nil {
		renderPSError("Part A – Q5: " + err.Error())
		return
	}

	// ── Part B (reuse existing getOptionMark) ────────────────────
	b1Mark, err := getOptionMark(r, "b1_option", "b1_mark", 16)
	if err != nil {
		renderPSError("Part B – Question 1: " + err.Error())
		return
	}
	b2Mark, err := getOptionMark(r, "b2_option", "b2_mark", 16)
	if err != nil {
		renderPSError("Part B – Question 2: " + err.Error())
		return
	}

	// ── Part C (reuse existing getOptionMark) ────────────────────
	c3Mark, err := getOptionMark(r, "c3_option", "c3_mark", 8)
	if err != nil {
		renderPSError("Part C – Question 3: " + err.Error())
		return
	}

	// ── Calculations (existing calculator package) ───────────────
	partA := calculator.CalculatePartATotal(q1, q2, q3, q4, q5)
	partB := calculator.CalculatePartBTotal(b1Mark, b2Mark)
	partC := calculator.CalculatePartCTotal(c3Mark)
	total := calculator.CalculateTotal(partA, partB, partC)
	pct := calculator.CalculatePercentage(total)

	student := models.Student{
		Name:       name,
		RegNo:      regNo,
		PartATotal: partA,
		PartBTotal: partB,
		PartCTotal: partC,
		Total:      total,
		Percentage: pct,
	}

	// ── Persist to shared in-memory store ────────────────────────
	// Same addStudent used by calculateHandler — student appears in
	// Dashboard and Class Analysis immediately.
	addStudent(student)

	// ── Redirect to home to show the result card ─────────────────
	// We redirect to "/" with the result encoded as query params so
	// the existing result card renders. However, since our current
	// home handler doesn't support query-param results, we render the
	// index template directly here with the result.
	updatedStats := calculator.ComputeClassStats(getStudents(), PASS_MARK)
	tmplIndex.Execute(w, IndexPageData{
		Result: &student,
		Stats:  updatedStats,
	})
}

// rebuildExtractionFromForm reconstructs a minimal ExtractionResult from
// form values so the verification page can be re-rendered with the faculty's
// current edits intact after a validation error.
func rebuildExtractionFromForm(r *http.Request) paperscan.ExtractionResult {
	markField := func(field string) paperscan.MarkField {
		raw := r.FormValue(field)
		v, err := strconv.Atoi(raw)
		status := paperscan.StatusNeedsVerification
		if err != nil || raw == "" {
			v = 0
			status = paperscan.StatusNotDetected
		}
		return paperscan.MarkField{Value: v, Status: status}
	}
	optField := func(field string) paperscan.OptionField {
		v := r.FormValue(field)
		status := paperscan.StatusNeedsVerification
		if v != "A" && v != "B" {
			v = ""
			status = paperscan.StatusNotDetected
		}
		return paperscan.OptionField{Value: v, Status: status}
	}
	return paperscan.ExtractionResult{
		Q1:         markField("q1"),
		Q2:         markField("q2"),
		Q3:         markField("q3"),
		Q4:         markField("q4"),
		Q5:         markField("q5"),
		B1Option:   optField("b1_option"),
		B1Mark:     markField("b1_mark"),
		B2Option:   optField("b2_option"),
		B2Mark:     markField("b2_mark"),
		C3Option:   optField("c3_option"),
		C3Mark:     markField("c3_mark"),
		EngineUsed: "demo",
		EngineNote: "Re-rendered after validation error.",
	}
}

// ─────────────────────────────────────────────
// VALIDATION HELPERS — UNCHANGED
// ─────────────────────────────────────────────

func getIntMark(r *http.Request, field string, max int) (int, error) {
	raw := r.FormValue(field)
	if raw == "" {
		return 0, fmt.Errorf("mark is required (0–%d)", max)
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("must be a whole number")
	}
	if v < 0 || v > max {
		return 0, fmt.Errorf("must be between 0 and %d", max)
	}
	return v, nil
}

func getOptionMark(r *http.Request, optionField, markField string, max int) (int, error) {
	option := r.FormValue(optionField)
	if option != "A" && option != "B" {
		return 0, fmt.Errorf("please select Option A or Option B")
	}
	return getIntMark(r, markField, max)
}
