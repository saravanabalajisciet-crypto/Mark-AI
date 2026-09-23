# MarkAI — Internal Marks Processing & Analysis System

> A clean, fast, single-binary web application for college faculty to enter, calculate, and analyse internal assessment marks.

---

## What it does

MarkAI handles the complete internal assessment workflow for a 50-mark exam paper:

- **Enter marks** for Part A (Q1–Q5), Part B (A/B options), and Part C (A/B option)
- **Calculate** part-wise totals and percentage automatically
- **Store** every submission in an in-memory class dataset
- **Analyse** class performance with live statistics and filters
- **PaperScan** — capture or upload the evaluated marks section and fill in the verification form before confirming

---

## Exam Structure

| Part | Questions | Max Marks |
|------|-----------|-----------|
| Part A | Q1–Q5 (2 marks each) | 10 |
| Part B | Q1 and Q2 (Option A or B, max 16 each) | 32 |
| Part C | Q3 (Option A or B, max 8) | 8 |
| **Total** | | **50** |

---

## Features

### Mark Entry
- Student name and register number
- Part A: five 0–2 mark inputs
- Part B / C: side-by-side Option A / Option B cards — selecting one disables the other
- Instant part-wise and grand total calculation
- Clean validation messages for every field

### Dashboard
- Live class statistics: Total Students · Passed · Needs Review · Class Average
- Honest empty state when no data has been entered yet
- Pass threshold configurable via `PASS_MARK` constant (currently 25 / 50)

### Class Analysis (`/class`)
- Summary cards: Total · Passed · Needs Review · Average · Highest · Lowest · Pass Rate
- Full student table with Part A / B / C breakdown, total, percentage, Pass / Review badge
- Filter pills: **All** · **Below 25** · **25–35** · **Above 35**

### PaperScan (`/paperscan`)
- **Camera capture** — uses the device rear camera; captures only the evaluated marks section
- **Image upload** — JPG / JPEG / PNG
- Both paths feed into the same extraction pipeline
- Extraction result shown with per-field confidence status: `Extracted` · `Needs Verification` · `Not Detected`
- Every field is editable before confirming
- **Confirm & Calculate** routes directly into the existing calculation engine — no duplicate logic
- Pluggable `Extractor` interface ready for a real OCR engine (Tesseract / EasyOCR) when available

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.21+ |
| HTTP server | `net/http` (standard library) |
| Templates | `html/template` (standard library) |
| Frontend | Vanilla HTML + CSS + JavaScript (no framework) |
| Storage | In-memory slice (session-scoped) |
| Build | Single binary — `go build` |

No external Go dependencies. No JavaScript framework. No database required to run.

---

## Getting Started

### Prerequisites

- [Go 1.21+](https://go.dev/dl/)

### Run locally

```bash
git clone https://github.com/saravanabalajisciet-crypto/Mark-AI.git
cd Mark-AI
go run main.go
```

Then open **http://localhost:8080** in your browser.

### Build a standalone binary

```bash
go build -o markai .
./markai          # Linux / macOS
.\markai.exe      # Windows
```

---

## Project Structure

```
Mark-AI/
├── main.go                  # HTTP server, all route handlers
├── go.mod
│
├── models/
│   └── student.go           # Student and ClassStats structs
│
├── calculator/
│   └── result.go            # CalculatePartATotal, ComputeClassStats, FilterStudents …
│
├── paperscan/
│   └── extractor.go         # Extractor interface, DemoExtractor, ExtractionResult
│
├── templates/
│   ├── index.html           # Dashboard + Mark Entry
│   ├── class.html           # Class Analysis
│   └── paperscan.html       # PaperScan upload / verify
│
└── static/
    └── style.css            # All styles — desktop + mobile responsive
```

---

## Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Dashboard |
| POST | `/calculate` | Submit and calculate marks |
| GET | `/class` | Class Analysis (optional `?filter=`) |
| GET | `/paperscan` | PaperScan upload page |
| POST | `/paperscan/extract` | Process uploaded/captured image |
| POST | `/paperscan/confirm` | Confirm verified marks → calculate |
| GET | `/static/*` | CSS and static assets |

---

## Mobile Support

MarkAI is optimised for **360 px – 430 px** phones:

- Fixed bottom navigation bar (Dashboard · Marks · Class · PaperScan)
- Touch-friendly inputs (44 px+ height, 16 px font to prevent iOS zoom)
- Camera preview fits phone screen with portrait crop
- Full-width Confirm & Calculate button
- Class table horizontally scrollable inside a contained area
- Desktop sidebar and layout unchanged above 768 px

---

## Configuration

Open `main.go` and change the constant:

```go
// Set to 25 for now. Update once the college confirms the actual threshold.
const PASS_MARK = 25
```

---

## Adding a Real OCR Engine (PaperScan)

The `paperscan.Extractor` interface is the integration point:

```go
type Extractor interface {
    Extract(file multipart.File, filename string) (ExtractionResult, error)
}
```

1. Implement `TesseractExtractor` (or any engine) in `paperscan/`
2. Return it from `NewExtractor()` when the binary is detected:

```go
func NewExtractor() Extractor {
    if _, err := exec.LookPath("tesseract"); err == nil {
        return TesseractExtractor{}
    }
    return DemoExtractor{} // transparent stub
}
```

The rest of the application requires no changes.

---

## Test Case

| Input | Expected |
|-------|----------|
| Part A: Q1=2 Q2=2 Q3=2 Q4=2 Q5=2 | Part A = 10 / 10 |
| Part B: Q1 → A → 10, Q2 → B → 12 | Part B = 22 / 32 |
| Part C: Q3 → A → 6 | Part C = 6 / 8 |
| | **Total = 38 / 50 · 76 %** |

---

## License

MIT
