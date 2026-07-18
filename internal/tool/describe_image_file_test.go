package tool

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePath_Relative(t *testing.T) {
	if err := validatePath("relative/path.png"); err == nil {
		t.Fatal("expected error for relative path")
	}
}

func TestValidatePath_Traversal(t *testing.T) {
	traversal := filepath.Join(string(filepath.Separator), "etc", "..", "etc", "passwd")
	if err := validatePath(traversal); err == nil {
		t.Fatal("expected error for path with ..")
	}
}

func TestValidatePath_TooLong(t *testing.T) {
	long := filepath.Join(string(filepath.Separator), "tmp", strings.Repeat("a", maxPathLength))
	if err := validatePath(long); err == nil {
		t.Fatal("expected error for over-long path")
	}
}

func TestValidatePath_AbsoluteOK(t *testing.T) {
	abs := filepath.Join(os.TempDir(), "image.png")
	if err := validatePath(abs); err != nil {
		t.Fatalf("unexpected error for valid absolute path: %v", err)
	}
}

func TestDescribeImageFile_Name(t *testing.T) {
	tool := NewDescribeImageFile(nil, nil)
	if tool.Name() != "describe_image_file" {
		t.Errorf("Name = %q", tool.Name())
	}
}

func TestDescribeImageFile_MissingArg(t *testing.T) {
	_, err := NewDescribeImageFile(nil, nil).Execute(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error when image_path missing")
	}
}
