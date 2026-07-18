package tool

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
)

func newBase64Tool() *DescribeBase64Image {
	return NewDescribeBase64Image(nil, nil)
}

func TestDescribeBase64_MissingArg(t *testing.T) {
	_, err := newBase64Tool().Execute(context.Background(), map[string]interface{}{})
	if err == nil || !strings.Contains(err.Error(), "缺失") {
		t.Fatalf("expected missing-arg error, got %v", err)
	}
}

func TestDescribeBase64_DataURIPrefix(t *testing.T) {
	args := map[string]interface{}{"image_base64": "data:image/png;base64,AAAA"}
	if _, err := newBase64Tool().Execute(context.Background(), args); err == nil {
		t.Fatal("expected error for data URI prefix")
	}
}

func TestDescribeBase64_InvalidBase64(t *testing.T) {
	args := map[string]interface{}{"image_base64": "!!!!not base64!!!"}
	if _, err := newBase64Tool().Execute(context.Background(), args); err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDescribeBase64_OversizeString(t *testing.T) {
	huge := strings.Repeat("A", maxBase64Len+1)
	args := map[string]interface{}{"image_base64": huge}
	if _, err := newBase64Tool().Execute(context.Background(), args); err == nil {
		t.Fatal("expected error for oversize base64 string")
	}
}

func TestDescribeBase64_OversizeDecoded(t *testing.T) {
	// ~21MB of zero bytes -> raw size > maxDecodedSize
	raw := make([]byte, 21*1024*1024)
	b64 := base64.StdEncoding.EncodeToString(raw)
	args := map[string]interface{}{"image_base64": b64}
	if _, err := newBase64Tool().Execute(context.Background(), args); err == nil {
		t.Fatal("expected error for oversize decoded data")
	}
}

func TestDescribeBase64_BadFormat(t *testing.T) {
	tiny := base64.StdEncoding.EncodeToString([]byte{0xFF, 0xD8, 0xFF, 0xE0})
	args := map[string]interface{}{"image_base64": tiny, "image_format": "gif"}
	if _, err := newBase64Tool().Execute(context.Background(), args); err == nil {
		t.Fatal("expected error for invalid image_format")
	}
}

func TestDescribeBase64_NameAndSchema(t *testing.T) {
	tool := newBase64Tool()
	if tool.Name() != "describe_base64_image" {
		t.Errorf("Name = %q", tool.Name())
	}
	schema := tool.InputSchema()
	req, _ := schema["required"].([]string)
	if len(req) == 0 || req[0] != "image_base64" {
		t.Errorf("required = %v, want image_base64", req)
	}
}
