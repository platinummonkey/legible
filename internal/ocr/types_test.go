package ocr

import (
	"testing"
)

func TestNewRectangle(t *testing.T) {
	rect := NewRectangle(10, 20, 100, 50)

	if rect.X != 10 {
		t.Errorf("expected X=10, got %d", rect.X)
	}
	if rect.Y != 20 {
		t.Errorf("expected Y=20, got %d", rect.Y)
	}
	if rect.Width != 100 {
		t.Errorf("expected Width=100, got %d", rect.Width)
	}
	if rect.Height != 50 {
		t.Errorf("expected Height=50, got %d", rect.Height)
	}
}

func TestRectangle_Area(t *testing.T) {
	rect := NewRectangle(0, 0, 100, 50)
	if rect.Area() != 5000 {
		t.Errorf("expected area 5000, got %d", rect.Area())
	}
}

func TestRectangle_Right(t *testing.T) {
	rect := NewRectangle(10, 20, 100, 50)
	if rect.Right() != 110 {
		t.Errorf("expected right edge 110, got %d", rect.Right())
	}
}

func TestRectangle_Bottom(t *testing.T) {
	rect := NewRectangle(10, 20, 100, 50)
	if rect.Bottom() != 70 {
		t.Errorf("expected bottom edge 70, got %d", rect.Bottom())
	}
}

func TestRectangle_Contains(t *testing.T) {
	rect := NewRectangle(10, 20, 100, 50)

	tests := []struct {
		name string
		x, y int
		want bool
	}{
		{"inside", 50, 40, true},
		{"top-left corner", 10, 20, true},
		{"bottom-right corner", 110, 70, true},
		{"outside left", 5, 40, false},
		{"outside top", 50, 15, false},
		{"outside right", 115, 40, false},
		{"outside bottom", 50, 75, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rect.Contains(tt.x, tt.y)
			if got != tt.want {
				t.Errorf("Contains(%d, %d) = %v, want %v", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func TestRectangle_Intersects(t *testing.T) {
	rect := NewRectangle(10, 10, 100, 100)

	tests := []struct {
		name  string
		other Rectangle
		want  bool
	}{
		{
			name:  "overlapping",
			other: NewRectangle(50, 50, 100, 100),
			want:  true,
		},
		{
			name:  "contained",
			other: NewRectangle(20, 20, 50, 50),
			want:  true,
		},
		{
			name:  "containing",
			other: NewRectangle(0, 0, 200, 200),
			want:  true,
		},
		{
			name:  "touching edge",
			other: NewRectangle(110, 10, 50, 100),
			want:  false,
		},
		{
			name:  "completely separate",
			other: NewRectangle(200, 200, 50, 50),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rect.Intersects(tt.other)
			if got != tt.want {
				t.Errorf("Intersects() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewWord(t *testing.T) {
	bbox := NewRectangle(10, 20, 50, 15)
	word := NewWord("hello", bbox, 95.5)

	if word.Text != "hello" {
		t.Errorf("expected text 'hello', got %s", word.Text)
	}
	if word.Confidence != 95.5 {
		t.Errorf("expected confidence 95.5, got %f", word.Confidence)
	}
	if word.BoundingBox.X != 10 {
		t.Errorf("expected bounding box X=10, got %d", word.BoundingBox.X)
	}
}

func TestNewPageOCR(t *testing.T) {
	page := NewPageOCR(1, 1404, 1872, "eng")

	if page.PageNumber != 1 {
		t.Errorf("expected page number 1, got %d", page.PageNumber)
	}
	if page.Width != 1404 {
		t.Errorf("expected width 1404, got %d", page.Width)
	}
	if page.Height != 1872 {
		t.Errorf("expected height 1872, got %d", page.Height)
	}
	if page.Language != "eng" {
		t.Errorf("expected language 'eng', got %s", page.Language)
	}
	if page.Words == nil {
		t.Error("Words should be initialized")
	}
}

func TestPageOCR_AddWord(t *testing.T) {
	page := NewPageOCR(1, 1404, 1872, "eng")

	word1 := NewWord("hello", NewRectangle(10, 20, 50, 15), 95.0)
	word2 := NewWord("world", NewRectangle(65, 20, 50, 15), 98.0)

	page.AddWord(word1)
	page.AddWord(word2)

	if len(page.Words) != 2 {
		t.Errorf("expected 2 words, got %d", len(page.Words))
	}
}

func TestPageOCR_CalculateConfidence(t *testing.T) {
	page := NewPageOCR(1, 1404, 1872, "eng")

	page.AddWord(NewWord("hello", NewRectangle(0, 0, 50, 15), 90.0))
	page.AddWord(NewWord("world", NewRectangle(0, 0, 50, 15), 100.0))
	page.AddWord(NewWord("test", NewRectangle(0, 0, 50, 15), 95.0))

	page.CalculateConfidence()

	expected := (90.0 + 100.0 + 95.0) / 3.0
	if page.Confidence != expected {
		t.Errorf("expected confidence %f, got %f", expected, page.Confidence)
	}
}

func TestPageOCR_CalculateConfidence_Empty(t *testing.T) {
	page := NewPageOCR(1, 1404, 1872, "eng")
	page.CalculateConfidence()

	if page.Confidence != 0 {
		t.Errorf("expected confidence 0 for empty page, got %f", page.Confidence)
	}
}

func TestPageOCR_BuildText(t *testing.T) {
	page := NewPageOCR(1, 1404, 1872, "eng")

	page.AddWord(NewWord("hello", NewRectangle(0, 0, 50, 15), 95.0))
	page.AddWord(NewWord("world", NewRectangle(0, 0, 50, 15), 98.0))
	page.AddWord(NewWord("test", NewRectangle(0, 0, 50, 15), 97.0))

	page.BuildText()

	expected := "hello world test"
	if page.Text != expected {
		t.Errorf("expected text %q, got %q", expected, page.Text)
	}
}

func TestPageOCR_BuildText_Empty(t *testing.T) {
	page := NewPageOCR(1, 1404, 1872, "eng")
	page.BuildText()

	if page.Text != "" {
		t.Errorf("expected empty text, got %q", page.Text)
	}
}

func TestNewDocumentOCR(t *testing.T) {
	doc := NewDocumentOCR("doc-123", "eng+fra")

	if doc.DocumentID != "doc-123" {
		t.Errorf("expected document ID 'doc-123', got %s", doc.DocumentID)
	}
	if doc.Language != "eng+fra" {
		t.Errorf("expected language 'eng+fra', got %s", doc.Language)
	}
	if doc.Pages == nil {
		t.Error("Pages should be initialized")
	}
}

func TestDocumentOCR_AddPage(t *testing.T) {
	doc := NewDocumentOCR("doc-123", "eng")

	page1 := *NewPageOCR(1, 1404, 1872, "eng")
	page2 := *NewPageOCR(2, 1404, 1872, "eng")

	doc.AddPage(page1)
	doc.AddPage(page2)

	if len(doc.Pages) != 2 {
		t.Errorf("expected 2 pages, got %d", len(doc.Pages))
	}
}

func TestDocumentOCR_Finalize(t *testing.T) {
	doc := NewDocumentOCR("doc-123", "eng")

	page1 := NewPageOCR(1, 1404, 1872, "eng")
	page1.AddWord(NewWord("hello", NewRectangle(0, 0, 50, 15), 90.0))
	page1.AddWord(NewWord("world", NewRectangle(0, 0, 50, 15), 95.0))
	page1.CalculateConfidence()

	page2 := NewPageOCR(2, 1404, 1872, "eng")
	page2.AddWord(NewWord("test", NewRectangle(0, 0, 50, 15), 100.0))
	page2.CalculateConfidence()

	doc.AddPage(*page1)
	doc.AddPage(*page2)
	doc.Finalize()

	if doc.TotalPages != 2 {
		t.Errorf("expected total pages 2, got %d", doc.TotalPages)
	}

	if doc.TotalWords != 3 {
		t.Errorf("expected total words 3, got %d", doc.TotalWords)
	}

	expectedConfidence := (page1.Confidence + page2.Confidence) / 2.0
	if doc.AverageConfidence != expectedConfidence {
		t.Errorf("expected average confidence %f, got %f", expectedConfidence, doc.AverageConfidence)
	}
}
