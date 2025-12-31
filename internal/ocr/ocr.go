package ocr

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/otiai10/gosseract/v2"
	"github.com/platinummonkey/remarkable-sync/internal/logger"
)

// Processor handles OCR processing using Tesseract
type Processor struct {
	logger    *logger.Logger
	languages []string
}

// Config holds configuration for the OCR processor
type Config struct {
	Logger    *logger.Logger
	Languages []string // Tesseract language codes (default: ["eng"])
}

// New creates a new OCR processor
func New(cfg *Config) *Processor {
	log := cfg.Logger
	if log == nil {
		log = logger.Get()
	}

	languages := cfg.Languages
	if len(languages) == 0 {
		languages = []string{"eng"}
	}

	return &Processor{
		logger:    log,
		languages: languages,
	}
}

// ProcessImage performs OCR on an image and returns structured results
func (p *Processor) ProcessImage(imageData []byte, pageNumber int) (*PageOCR, error) {
	p.logger.WithFields("page", pageNumber, "image_size", len(imageData)).Debug("Processing image with OCR")

	startTime := time.Now()

	// Create Tesseract client
	client := gosseract.NewClient()
	defer client.Close()

	// Configure languages
	if err := client.SetLanguage(p.languages...); err != nil {
		return nil, fmt.Errorf("failed to set OCR language: %w", err)
	}

	// Set image data
	if err := client.SetImageFromBytes(imageData); err != nil {
		return nil, fmt.Errorf("failed to set image data: %w", err)
	}

	// Get HOCR output (HTML-based OCR with position information)
	hocrText, err := client.HOCRText()
	if err != nil {
		return nil, fmt.Errorf("failed to get HOCR text: %w", err)
	}

	// Parse HOCR to extract words with bounding boxes
	pageOCR, err := p.parseHOCR(hocrText, pageNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HOCR: %w", err)
	}

	// Build full text and calculate confidence
	pageOCR.BuildText()
	pageOCR.CalculateConfidence()

	duration := time.Since(startTime)
	p.logger.WithFields(
		"page", pageNumber,
		"words", len(pageOCR.Words),
		"confidence", pageOCR.Confidence,
		"duration", duration,
	).Info("OCR processing completed")

	return pageOCR, nil
}

// parseHOCR parses HOCR XML output to extract words with bounding boxes
func (p *Processor) parseHOCR(hocrText string, pageNumber int) (*PageOCR, error) {
	// Parse HOCR XML
	var page HOCRPage
	if err := xml.Unmarshal([]byte(hocrText), &page); err != nil {
		return nil, fmt.Errorf("failed to unmarshal HOCR XML: %w", err)
	}

	// Extract page dimensions from the ocr_page div
	width, height := 0, 0
	if len(page.Body.Pages) > 0 {
		if bbox := extractBBox(page.Body.Pages[0].Title); len(bbox) >= 4 {
			width = bbox[2]
			height = bbox[3]
		}
	}

	pageOCR := NewPageOCR(pageNumber, width, height, strings.Join(p.languages, "+"))

	// Extract words from all content areas in all page divs
	for _, pageDiv := range page.Body.Pages {
		for _, area := range pageDiv.Areas {
			for _, par := range area.Pars {
				for _, line := range par.Lines {
					for _, word := range line.Words {
						bbox := extractBBox(word.Title)
						if len(bbox) >= 4 {
							confidence := extractConfidence(word.Title)

							w := NewWord(
								strings.TrimSpace(word.Text),
								NewRectangle(bbox[0], bbox[1], bbox[2]-bbox[0], bbox[3]-bbox[1]),
								confidence,
							)
							pageOCR.AddWord(w)
						}
					}
				}
			}
		}
	}

	return pageOCR, nil
}

// extractBBox extracts bounding box coordinates from HOCR title attribute
// Format: "bbox x0 y0 x1 y1" or "bbox x0 y0 x1 y1; x_wconf 95"
func extractBBox(title string) []int {
	re := regexp.MustCompile(`bbox\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+)`)
	matches := re.FindStringSubmatch(title)

	if len(matches) != 5 {
		return nil
	}

	bbox := make([]int, 4)
	for i := 0; i < 4; i++ {
		val, err := strconv.Atoi(matches[i+1])
		if err != nil {
			return nil
		}
		bbox[i] = val
	}

	return bbox
}

// extractConfidence extracts confidence score from HOCR title attribute
// Format: "bbox x0 y0 x1 y1; x_wconf 95"
func extractConfidence(title string) float64 {
	re := regexp.MustCompile(`x_wconf\s+(\d+)`)
	matches := re.FindStringSubmatch(title)

	if len(matches) != 2 {
		return 0.0
	}

	conf, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0.0
	}

	return conf
}

// HOCRPage represents the HOCR XML structure
type HOCRPage struct {
	XMLName xml.Name `xml:"html"`
	Title   string   `xml:"head>title"`
	Body    HOCRBody `xml:"body"`
}

// HOCRBody represents the body section of HOCR
type HOCRBody struct {
	Pages []HOCRPageDiv `xml:"div"`
}

// HOCRPageDiv represents an ocr_page div (page container)
type HOCRPageDiv struct {
	Class string     `xml:"class,attr"`
	Title string     `xml:"title,attr"`
	Areas []HOCRArea `xml:"div"`
}

// HOCRArea represents an ocr_carea (content area)
type HOCRArea struct {
	Class string    `xml:"class,attr"`
	Title string    `xml:"title,attr"`
	Pars  []HOCRPar `xml:"p"`
}

// HOCRPar represents an ocr_par (paragraph)
type HOCRPar struct {
	Class string     `xml:"class,attr"`
	Title string     `xml:"title,attr"`
	Lines []HOCRLine `xml:"span"`
}

// HOCRLine represents an ocr_line (text line)
type HOCRLine struct {
	Class string     `xml:"class,attr"`
	Title string     `xml:"title,attr"`
	Words []HOCRWord `xml:"span"`
}

// HOCRWord represents an ocr_word (individual word)
type HOCRWord struct {
	Class string `xml:"class,attr"`
	Title string `xml:"title,attr"`
	Text  string `xml:",chardata"`
}
