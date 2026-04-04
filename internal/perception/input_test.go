package perception

import (
	"mime/multipart"
	"net/textproto"
	"strings"
	"testing"
)

func TestMediaTypeDetector_DetectFromHeader_Text(t *testing.T) {
	d := NewMediaTypeDetector()
	data := []byte("Hello, this is plain text.")
	if mt := d.DetectFromHeader(data); mt != MediaTypeText {
		t.Fatalf("expected text, got %s", mt)
	}
}

func TestMediaTypeDetector_DetectFromHeader_PDF(t *testing.T) {
	d := NewMediaTypeDetector()
	data := []byte("%PDF-1.4 some pdf content")
	if mt := d.DetectFromHeader(data); mt != MediaTypePDF {
		t.Fatalf("expected pdf, got %s", mt)
	}
}

func TestMediaTypeDetector_DetectFromHeader_PNG(t *testing.T) {
	d := NewMediaTypeDetector()
	data := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	if mt := d.DetectFromHeader(data); mt != MediaTypeImage {
		t.Fatalf("expected image, got %s", mt)
	}
}

func TestMediaTypeDetector_DetectFromHeader_JPEG(t *testing.T) {
	d := NewMediaTypeDetector()
	data := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	if mt := d.DetectFromHeader(data); mt != MediaTypeImage {
		t.Fatalf("expected image, got %s", mt)
	}
}

func TestMediaTypeDetector_DetectFromHeader_MP3(t *testing.T) {
	d := NewMediaTypeDetector()
	data := []byte("ID3\x04\x00\x00")
	if mt := d.DetectFromHeader(data); mt != MediaTypeAudio {
		t.Fatalf("expected audio, got %s", mt)
	}
}

func TestMediaTypeDetector_DetectFromHeader_WAV(t *testing.T) {
	d := NewMediaTypeDetector()
	data := []byte("RIFF\x00\x00\x00\x00WAVE")
	if mt := d.DetectFromHeader(data); mt != MediaTypeAudio {
		t.Fatalf("expected audio, got %s", mt)
	}
}

func TestMediaTypeDetector_DetectFromHeader_OGG(t *testing.T) {
	d := NewMediaTypeDetector()
	data := []byte("OggS\x00\x00\x00\x00")
	if mt := d.DetectFromHeader(data); mt != MediaTypeAudio {
		t.Fatalf("expected audio, got %s", mt)
	}
}

func TestMediaTypeDetector_DetectFromMIME(t *testing.T) {
	d := NewMediaTypeDetector()
	tests := []struct {
		mime     string
		expected MediaType
	}{
		{"text/plain", MediaTypeText},
		{"image/png", MediaTypeImage},
		{"audio/mpeg", MediaTypeAudio},
		{"application/pdf", MediaTypePDF},
		{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", MediaTypeDOCX},
		{"application/octet-stream", MediaTypeUnknown},
	}
	for _, tt := range tests {
		if mt := d.DetectFromMIME(tt.mime); mt != tt.expected {
			t.Errorf("DetectFromMIME(%q) = %s, want %s", tt.mime, mt, tt.expected)
		}
	}
}

func TestMediaTypeDetector_DetectFromExtension(t *testing.T) {
	d := NewMediaTypeDetector()
	tests := []struct {
		filename string
		expected MediaType
	}{
		{"report.pdf", MediaTypePDF},
		{"photo.jpg", MediaTypeImage},
		{"song.mp3", MediaTypeAudio},
		{"document.docx", MediaTypeDOCX},
		{"notes.txt", MediaTypeText},
		{"data.csv", MediaTypeText},
		{"unknown.xyz", MediaTypeUnknown},
	}
	for _, tt := range tests {
		if mt := d.DetectFromExtension(tt.filename); mt != tt.expected {
			t.Errorf("DetectFromExtension(%q) = %s, want %s", tt.filename, mt, tt.expected)
		}
	}
}

func TestMediaTypeDetector_DetectFromMultipart(t *testing.T) {
	d := NewMediaTypeDetector()

	// Create a mock multipart file header
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", "image/png")
	fileHeader := &multipart.FileHeader{
		Filename: "test.png",
		Header:   header,
	}

	mt, err := d.DetectFromMultipart(fileHeader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mt != MediaTypeImage {
		t.Fatalf("expected image, got %s", mt)
	}
}

func TestIsLikelyText(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{"empty", []byte{}, true},
		{"plain text", []byte("Hello, world!"), true},
		{"binary", []byte{0x00, 0x01, 0x02, 0x03, 0x04}, false},
		{"mixed", []byte("Hello\x00\x01\x02"), false},
		{"with newlines", []byte("line1\nline2\nline3"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLikelyText(tt.data)
			if result != tt.expected {
				t.Errorf("isLikelyText(%q) = %v, want %v", tt.data, result, tt.expected)
			}
		})
	}
}

func TestParseInput_Text(t *testing.T) {
	d := NewMediaTypeDetector()

	// Create a minimal multipart form with a text file
	var buf strings.Builder
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "hello.txt")
	part.Write([]byte("Hello, world!"))
	writer.Close()

	form, err := multipart.NewReader(strings.NewReader(buf.String()), writer.Boundary()).ReadForm(1024)
	if err != nil {
		t.Fatalf("failed to read form: %v", err)
	}
	defer form.RemoveAll()

	result, err := ParseInput(form, d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello, world!" {
		t.Errorf("expected 'Hello, world!', got %q", result)
	}
}

func TestParseInput_ImagePlaceholder(t *testing.T) {
	d := NewMediaTypeDetector()

	var buf strings.Builder
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "photo.png")
	// Write PNG magic bytes
	part.Write([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0x00})
	writer.Close()

	form, err := multipart.NewReader(strings.NewReader(buf.String()), writer.Boundary()).ReadForm(1024)
	if err != nil {
		t.Fatalf("failed to read form: %v", err)
	}
	defer form.RemoveAll()

	result, err := ParseInput(form, d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "[IMAGE]") {
		t.Errorf("expected image placeholder, got %q", result)
	}
}

func TestParseInput_PDFPlaceholder(t *testing.T) {
	d := NewMediaTypeDetector()

	var buf strings.Builder
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "doc.pdf")
	part.Write([]byte("%PDF-1.4 content"))
	writer.Close()

	form, err := multipart.NewReader(strings.NewReader(buf.String()), writer.Boundary()).ReadForm(1024)
	if err != nil {
		t.Fatalf("failed to read form: %v", err)
	}
	defer form.RemoveAll()

	result, err := ParseInput(form, d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "[PDF]") {
		t.Errorf("expected PDF placeholder, got %q", result)
	}
}

func TestParseInput_NilForm(t *testing.T) {
	d := NewMediaTypeDetector()
	result, err := ParseInput(nil, d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestConvenienceFunctions(t *testing.T) {
	if mt := DetectContentType([]byte("hello")); mt != MediaTypeText {
		t.Errorf("DetectContentType(text) = %s, want text", mt)
	}
	if mt := DetectContentTypeFromMIME("image/jpeg"); mt != MediaTypeImage {
		t.Errorf("DetectContentTypeFromMIME(image/jpeg) = %s, want image", mt)
	}
	if mt := DetectContentTypeFromFilename("test.mp3"); mt != MediaTypeAudio {
		t.Errorf("DetectContentTypeFromFilename(test.mp3) = %s, want audio", mt)
	}
}
