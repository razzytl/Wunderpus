package perception

import (
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
)

// MediaType represents the detected type of an input payload.
type MediaType string

const (
	MediaTypeText    MediaType = "text"
	MediaTypeImage   MediaType = "image"
	MediaTypeAudio   MediaType = "audio"
	MediaTypePDF     MediaType = "pdf"
	MediaTypeDOCX    MediaType = "docx"
	MediaTypeUnknown MediaType = "unknown"
)

// MediaTypeDetector identifies the type of input data from file headers,
// extensions, or MIME types, and routes it to the appropriate parser.
type MediaTypeDetector struct{}

// NewMediaTypeDetector creates a new detector.
func NewMediaTypeDetector() *MediaTypeDetector {
	return &MediaTypeDetector{}
}

// DetectFromHeader examines the first bytes of data to determine the media type.
// Uses magic number signatures for common file formats.
func (d *MediaTypeDetector) DetectFromHeader(data []byte) MediaType {
	if len(data) == 0 {
		return MediaTypeText
	}

	// Check magic numbers FIRST (before text heuristic, since PDF/ZIP start with printable ASCII)

	// PDF: %PDF
	if len(data) >= 4 && string(data[:4]) == "%PDF" {
		return MediaTypePDF
	}

	// PNG: 89 50 4E 47
	if len(data) >= 4 && data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
		return MediaTypeImage
	}

	// JPEG: FF D8 FF
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return MediaTypeImage
	}

	// GIF: GIF8
	if len(data) >= 4 && string(data[:4]) == "GIF8" {
		return MediaTypeImage
	}

	// WEBP: RIFF....WEBP
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return MediaTypeImage
	}

	// DOCX: PK (ZIP-based, same as xlsx/pptx)
	if len(data) >= 2 && data[0] == 'P' && data[1] == 'K' {
		// Could be DOCX, XLSX, PPTX, or any ZIP-based format
		// We'll classify as docx for now; a more precise check would inspect
		// the ZIP central directory for word/document.xml
		return MediaTypeDOCX
	}

	// OGG audio: OggS
	if len(data) >= 4 && string(data[:4]) == "OggS" {
		return MediaTypeAudio
	}

	// WAV: RIFF....WAVE
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WAVE" {
		return MediaTypeAudio
	}

	// MP3: ID3 or FF FB
	if len(data) >= 3 && (string(data[:3]) == "ID3" || (data[0] == 0xFF && data[1] == 0xFB)) {
		return MediaTypeAudio
	}

	// FLAC: fLaC
	if len(data) >= 4 && string(data[:4]) == "fLaC" {
		return MediaTypeAudio
	}

	// Check for text (printable ASCII/UTF8) — only after all magic numbers
	if isLikelyText(data) {
		return MediaTypeText
	}

	return MediaTypeUnknown
}

// DetectFromMIME determines media type from a MIME type string.
func (d *MediaTypeDetector) DetectFromMIME(mimeType string) MediaType {
	switch {
	case strings.HasPrefix(mimeType, "text/"):
		return MediaTypeText
	case strings.HasPrefix(mimeType, "image/"):
		return MediaTypeImage
	case strings.HasPrefix(mimeType, "audio/"):
		return MediaTypeAudio
	case mimeType == "application/pdf":
		return MediaTypePDF
	case mimeType == "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return MediaTypeDOCX
	default:
		return MediaTypeUnknown
	}
}

// DetectFromExtension determines media type from a file extension.
func (d *MediaTypeDetector) DetectFromExtension(filename string) MediaType {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".txt", ".md", ".log", ".csv", ".json", ".xml", ".yaml", ".yml", ".html", ".htm", ".rtf":
		return MediaTypeText
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg", ".tiff", ".tif", ".ico":
		return MediaTypeImage
	case ".mp3", ".wav", ".ogg", ".flac", ".aac", ".m4a", ".opus", ".wma":
		return MediaTypeAudio
	case ".pdf":
		return MediaTypePDF
	case ".docx":
		return MediaTypeDOCX
	default:
		return MediaTypeUnknown
	}
}

// DetectFromMultipart detects the media type from an uploaded multipart form file.
func (d *MediaTypeDetector) DetectFromMultipart(file *multipart.FileHeader) (MediaType, error) {
	// Try MIME type from the form
	if file.Header != nil {
		if ct := file.Header.Get("Content-Type"); ct != "" {
			mt := d.DetectFromMIME(ct)
			if mt != MediaTypeUnknown {
				return mt, nil
			}
		}
	}

	// Try extension
	mt := d.DetectFromExtension(file.Filename)
	if mt != MediaTypeUnknown {
		return mt, nil
	}

	// Open and read header bytes
	f, err := file.Open()
	if err != nil {
		return MediaTypeUnknown, err
	}
	defer f.Close()

	header := make([]byte, 512)
	n, _ := f.Read(header)
	return d.DetectFromHeader(header[:n]), nil
}

// isLikelyText checks if data looks like text (mostly printable ASCII/UTF8).
func isLikelyText(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	printable := 0
	sampleSize := len(data)
	if sampleSize > 512 {
		sampleSize = 512
	}
	for _, b := range data[:sampleSize] {
		// Allow common whitespace and printable ASCII
		if (b >= 32 && b <= 126) || b == '\n' || b == '\r' || b == '\t' {
			printable++
		}
	}
	// If > 90% printable, it's likely text
	return float64(printable)/float64(sampleSize) > 0.9
}

// ParseInput is the main entry point: takes a multipart form, detects the type,
// and returns extracted text. For text, it returns the content as-is.
// For images, audio, PDF, DOCX it returns a placeholder describing the integration point.
func ParseInput(form *multipart.Form, detector *MediaTypeDetector) (string, error) {
	if form == nil || len(form.File) == 0 {
		return "", nil
	}

	// Get the first file from the form
	var fileHeader *multipart.FileHeader
	for _, files := range form.File {
		if len(files) > 0 {
			fileHeader = files[0]
			break
		}
	}
	if fileHeader == nil {
		return "", nil
	}

	mediaType, err := detector.DetectFromMultipart(fileHeader)
	if err != nil {
		return "", fmt.Errorf("perception: detecting media type: %w", err)
	}

	f, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("perception: opening uploaded file: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("perception: reading uploaded file: %w", err)
	}

	switch mediaType {
	case MediaTypeText:
		return string(data), nil
	case MediaTypeImage:
		// Integration point: Vision model call (e.g., gpt-4o with image_url)
		// TODO: Integrate with provider that supports vision (OpenAI, Gemini)
		// Example: call provider with Message{MultiContent: []ContentPart{{Type: "image_url", ImageURL: &ImageURL{URL: base64Data}}}}
		return "[IMAGE] Vision model integration required — connect gpt-4o or gemini-2.0-flash-vision for image-to-text", nil
	case MediaTypeAudio:
		// Integration point: Whisper API or whisper.cpp binding
		// TODO: Integrate OpenAI Whisper API or local whisper.cpp
		// Example: POST https://api.openai.com/v1/audio/transcriptions with file + model="whisper-1"
		return "[AUDIO] Whisper transcription required — connect OpenAI Whisper API or local whisper.cpp for audio-to-text", nil
	case MediaTypePDF:
		// Integration point: pdfcpu or unioffice for PDF text extraction
		// TODO: Integrate github.com/pdfcpu/pdfcpu or github.com/unidoc/unipdf/v3
		return "[PDF] PDF parsing required — integrate pdfcpu or unipdf for text extraction", nil
	case MediaTypeDOCX:
		// Integration point: github.com/nguyenthenguyen/docx or unioffice
		// TODO: Integrate a DOCX parser for text extraction
		return "[DOCX] DOCX parsing required — integrate a DOCX reader for text extraction", nil
	default:
		return string(data), nil // fallback: treat as text
	}
}

// DetectContentType is a convenience function that detects type from raw bytes.
func DetectContentType(data []byte) MediaType {
	d := NewMediaTypeDetector()
	return d.DetectFromHeader(data)
}

// DetectContentTypeFromMIME is a convenience function that detects type from MIME string.
func DetectContentTypeFromMIME(mimeType string) MediaType {
	d := NewMediaTypeDetector()
	return d.DetectFromMIME(mimeType)
}

// DetectContentTypeFromFilename is a convenience function that detects type from filename.
func DetectContentTypeFromFilename(filename string) MediaType {
	d := NewMediaTypeDetector()
	return d.DetectFromExtension(filename)
}

// TextToSpeechPayload holds parameters for a text-to-speech request.
type TextToSpeechPayload struct {
	Text  string `json:"text"`
	Voice string `json:"voice,omitempty"` // e.g., "alloy", "echo", "fable"
	Model string `json:"model,omitempty"` // e.g., "tts-1", "tts-1-hd"
}

// SpeechToTextPayload holds parameters for a speech-to-text request.
type SpeechToTextPayload struct {
	AudioData []byte `json:"audio_data"`
	Format    string `json:"format"`   // "mp3", "wav", "ogg", "flac"
	Model     string `json:"model"`    // "whisper-1" or local model path
	Language  string `json:"language"` // optional: "en", "fr", etc.
}

// ImageToTextPayload holds parameters for a vision model request.
type ImageToTextPayload struct {
	ImageData []byte `json:"image_data"`
	MIMEType  string `json:"mime_type"` // "image/png", "image/jpeg", etc.
	Prompt    string `json:"prompt"`    // optional: "Describe this image"
	Model     string `json:"model"`     // "gpt-4o", "gemini-2.0-flash", etc.
}
