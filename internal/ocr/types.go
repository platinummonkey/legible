package ocr

// PageOCR represents OCR results for a single page
type PageOCR struct {
	// PageNumber is the page number (1-indexed)
	PageNumber int

	// Words contains all recognized words on the page with their positions
	Words []Word

	// Text is the full text content of the page (for convenience)
	Text string

	// Confidence is the overall confidence score for the page (0-100)
	Confidence float64

	// Width is the page width in pixels
	Width int

	// Height is the page height in pixels
	Height int

	// Language is the detected or configured language
	Language string
}

// Word represents a single recognized word with its bounding box
type Word struct {
	// Text is the recognized text content
	Text string

	// BoundingBox is the position and size of the word on the page
	BoundingBox Rectangle

	// Confidence is the recognition confidence score (0-100)
	Confidence float64

	// FontSize is the estimated font size in points
	FontSize float64

	// Bold indicates if the word appears to be bold
	Bold bool

	// Italic indicates if the word appears to be italic
	Italic bool
}

// Rectangle represents a rectangular bounding box
type Rectangle struct {
	// X is the left coordinate (pixels from left edge)
	X int

	// Y is the top coordinate (pixels from top edge)
	Y int

	// Width is the width of the rectangle in pixels
	Width int

	// Height is the height of the rectangle in pixels
	Height int
}

// Line represents a line of text (multiple words)
type Line struct {
	// Words contains the words in this line
	Words []Word

	// BoundingBox is the bounding box for the entire line
	BoundingBox Rectangle

	// Text is the concatenated text of all words in the line
	Text string

	// Confidence is the average confidence of all words in the line
	Confidence float64
}

// Paragraph represents a paragraph (multiple lines)
type Paragraph struct {
	// Lines contains the lines in this paragraph
	Lines []Line

	// BoundingBox is the bounding box for the entire paragraph
	BoundingBox Rectangle

	// Text is the concatenated text of all lines in the paragraph
	Text string

	// Confidence is the average confidence of all lines in the paragraph
	Confidence float64
}

// DocumentOCR represents OCR results for an entire document
type DocumentOCR struct {
	// DocumentID is the unique identifier for the document
	DocumentID string

	// Pages contains OCR results for each page
	Pages []PageOCR

	// TotalPages is the total number of pages processed
	TotalPages int

	// TotalWords is the total number of words recognized
	TotalWords int

	// AverageConfidence is the average confidence across all pages
	AverageConfidence float64

	// ProcessingTime is the time taken to process the document (in seconds)
	ProcessingTime float64

	// Language is the OCR language(s) used
	Language string
}

// OCRResult represents the result of an OCR operation
type OCRResult struct {
	// DocumentOCR contains the OCR results
	DocumentOCR *DocumentOCR

	// Success indicates if OCR completed successfully
	Success bool

	// Error contains any error message if Success is false
	Error string
}

// NewRectangle creates a new Rectangle
func NewRectangle(x, y, width, height int) Rectangle {
	return Rectangle{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
	}
}

// Area returns the area of the rectangle
func (r Rectangle) Area() int {
	return r.Width * r.Height
}

// Right returns the right edge coordinate
func (r Rectangle) Right() int {
	return r.X + r.Width
}

// Bottom returns the bottom edge coordinate
func (r Rectangle) Bottom() int {
	return r.Y + r.Height
}

// Contains returns true if the rectangle contains the point (x, y)
func (r Rectangle) Contains(x, y int) bool {
	return x >= r.X && x <= r.Right() && y >= r.Y && y <= r.Bottom()
}

// Intersects returns true if this rectangle intersects with another
func (r Rectangle) Intersects(other Rectangle) bool {
	return r.X < other.Right() &&
		r.Right() > other.X &&
		r.Y < other.Bottom() &&
		r.Bottom() > other.Y
}

// NewWord creates a new Word
func NewWord(text string, bbox Rectangle, confidence float64) Word {
	return Word{
		Text:        text,
		BoundingBox: bbox,
		Confidence:  confidence,
	}
}

// NewPageOCR creates a new PageOCR result
func NewPageOCR(pageNumber, width, height int, language string) *PageOCR {
	return &PageOCR{
		PageNumber: pageNumber,
		Words:      []Word{},
		Width:      width,
		Height:     height,
		Language:   language,
	}
}

// AddWord adds a word to the page OCR result
func (p *PageOCR) AddWord(word Word) {
	p.Words = append(p.Words, word)
}

// CalculateConfidence calculates the average confidence for the page
func (p *PageOCR) CalculateConfidence() {
	if len(p.Words) == 0 {
		p.Confidence = 0
		return
	}

	total := 0.0
	for _, word := range p.Words {
		total += word.Confidence
	}
	p.Confidence = total / float64(len(p.Words))
}

// BuildText concatenates all word text to build the full page text
func (p *PageOCR) BuildText() {
	if len(p.Words) == 0 {
		p.Text = ""
		return
	}

	text := ""
	for i, word := range p.Words {
		if i > 0 {
			text += " "
		}
		text += word.Text
	}
	p.Text = text
}

// NewDocumentOCR creates a new DocumentOCR
func NewDocumentOCR(documentID string, language string) *DocumentOCR {
	return &DocumentOCR{
		DocumentID: documentID,
		Pages:      []PageOCR{},
		Language:   language,
	}
}

// AddPage adds a page to the document OCR results
func (d *DocumentOCR) AddPage(page PageOCR) {
	d.Pages = append(d.Pages, page)
}

// Finalize calculates summary statistics after all pages are processed
func (d *DocumentOCR) Finalize() {
	d.TotalPages = len(d.Pages)

	totalWords := 0
	totalConfidence := 0.0

	for _, page := range d.Pages {
		totalWords += len(page.Words)
		totalConfidence += page.Confidence
	}

	d.TotalWords = totalWords
	if d.TotalPages > 0 {
		d.AverageConfidence = totalConfidence / float64(d.TotalPages)
	}
}
