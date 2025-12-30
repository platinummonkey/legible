package rmclient

import "time"

// Document represents a reMarkable document or notebook
type Document struct {
	// ID is the unique identifier for this document
	ID string

	// Version is the document version number
	Version int

	// Name is the user-visible name of the document
	Name string

	// ModifiedClient is when the document was last modified on the client
	ModifiedClient time.Time

	// Type is the document type (DocumentType or CollectionType)
	Type string

	// CurrentPage is the currently viewed page (for documents)
	CurrentPage int

	// Parent is the ID of the parent collection (empty string for root)
	Parent string
}

// Metadata represents document metadata from .metadata files
type Metadata struct {
	// Deleted indicates if the document has been deleted
	Deleted bool `json:"deleted"`

	// LastModified is when the metadata was last modified
	LastModified string `json:"lastModified"`

	// LastOpened is when the document was last opened
	LastOpened string `json:"lastOpened"`

	// LastOpenedPage is the page number that was last opened
	LastOpenedPage int `json:"lastOpenedPage"`

	// MetadataModified indicates if metadata has been modified
	MetadataModified bool `json:"metadatamodified"`

	// Modified indicates if the document has been modified
	Modified bool `json:"modified"`

	// Parent is the ID of the parent collection
	Parent string `json:"parent"`

	// Pinned indicates if the document is pinned
	Pinned bool `json:"pinned"`

	// Synced indicates if the document has been synced
	Synced bool `json:"synced"`

	// Type is the document type ("DocumentType" or "CollectionType")
	Type string `json:"type"`

	// Version is the metadata version
	Version int `json:"version"`

	// VisibleName is the user-visible name
	VisibleName string `json:"visibleName"`
}

// Collection represents a reMarkable collection (folder)
type Collection struct {
	// ID is the unique identifier for this collection
	ID string

	// Name is the user-visible name of the collection
	Name string

	// Parent is the ID of the parent collection (empty string for root)
	Parent string

	// Type is always "CollectionType"
	Type string

	// ModifiedClient is when the collection was last modified
	ModifiedClient time.Time

	// Documents contains the documents in this collection
	Documents []Document

	// SubCollections contains nested collections
	SubCollections []Collection
}

// Content represents the .content file structure
type Content struct {
	// ExtraMetadata contains additional metadata
	ExtraMetadata ExtraMetadata `json:"extraMetadata"`

	// FileType is the type of file (e.g., "notebook", "pdf")
	FileType string `json:"fileType"`

	// FontName is the font name for text
	FontName string `json:"fontName"`

	// LastOpenedPage is the page number that was last opened
	LastOpenedPage int `json:"lastOpenedPage"`

	// LineHeight is the line height for text
	LineHeight int `json:"lineHeight"`

	// Margins is the page margins
	Margins int `json:"margins"`

	// Orientation is the page orientation (portrait/landscape)
	Orientation string `json:"orientation"`

	// PageCount is the number of pages
	PageCount int `json:"pageCount"`

	// Pages contains the page UUIDs
	Pages []string `json:"pages"`

	// TextScale is the text scaling factor
	TextScale int `json:"textScale"`

	// Transform contains transformation data
	Transform Transform `json:"transform"`
}

// ExtraMetadata contains additional document metadata
type ExtraMetadata struct {
	// LastBrushColor is the color of the last brush used
	LastBrushColor string `json:"LastBrushColor"`

	// LastBrushThicknessScale is the thickness scale of the last brush
	LastBrushThicknessScale string `json:"LastBrushThicknessScale"`

	// LastColor is the last color used
	LastColor string `json:"LastColor"`

	// LastEraserThicknessScale is the thickness scale of the last eraser
	LastEraserThicknessScale string `json:"LastEraserThicknessScale"`

	// LastEraserTool is the last eraser tool used
	LastEraserTool string `json:"LastEraserTool"`

	// LastPen is the last pen used
	LastPen string `json:"LastPen"`

	// LastPenColor is the color of the last pen
	LastPenColor string `json:"LastPenColor"`

	// LastPenThicknessScale is the thickness scale of the last pen
	LastPenThicknessScale string `json:"LastPenThicknessScale"`

	// LastPencil is the last pencil used
	LastPencil string `json:"LastPencil"`

	// LastPencilColor is the color of the last pencil
	LastPencilColor string `json:"LastPencilColor"`

	// LastPencilThicknessScale is the thickness scale of the last pencil
	LastPencilThicknessScale string `json:"LastPencilThicknessScale"`

	// LastTool is the last tool used
	LastTool string `json:"LastTool"`

	// ThicknessScale is the general thickness scale
	ThicknessScale string `json:"ThicknessScale"`
}

// Transform contains transformation data for pages
type Transform struct {
	// M11, M12, M13 are transformation matrix elements
	M11 float64 `json:"m11"`
	M12 float64 `json:"m12"`
	M13 float64 `json:"m13"`

	// M21, M22, M23 are transformation matrix elements
	M21 float64 `json:"m21"`
	M22 float64 `json:"m22"`
	M23 float64 `json:"m23"`

	// M31, M32, M33 are transformation matrix elements
	M31 float64 `json:"m31"`
	M32 float64 `json:"m32"`
	M33 float64 `json:"m33"`
}

// DocumentType constants
const (
	DocumentType   = "DocumentType"
	CollectionType = "CollectionType"
)

// FileType constants
const (
	FileTypeNotebook = "notebook"
	FileTypePDF      = "pdf"
	FileTypeEPUB     = "epub"
)
