package tool

import (
	"context"
	"testing"
)

type mockTool struct {
	name        string
	description string
	schema      map[string]interface{}
	result      string
	err         error
}

func (m *mockTool) Name() string                        { return m.name }
func (m *mockTool) Description() string                 { return m.description }
func (m *mockTool) InputSchema() map[string]interface{} { return m.schema }
func (m *mockTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	return m.result, m.err
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	m := &mockTool{name: "test_tool"}
	r.Register(m)

	got, ok := r.Get("test_tool")
	if !ok {
		t.Fatal("tool not found")
	}
	if got.Name() != "test_tool" {
		t.Errorf("got %q, want test_tool", got.Name())
	}

	_, ok = r.Get("nonexistent")
	if ok {
		t.Error("should not find nonexistent tool")
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockTool{name: "tool1"})
	r.Register(&mockTool{name: "tool2"})

	list := r.List()
	if len(list) != 2 {
		t.Errorf("expected 2 tools, got %d", len(list))
	}
}

func TestSanitizeArgs(t *testing.T) {
	args := map[string]interface{}{
		"image_path": "/test.png",
		"_meta":      map[string]interface{}{"current_model": "deepseek"},
	}
	clean := SanitizeArgs(args)
	if _, ok := clean["_meta"]; ok {
		t.Error("_meta should be removed")
	}
	if _, ok := clean["image_path"]; !ok {
		t.Error("image_path should be present")
	}
}

func TestValidateRequired(t *testing.T) {
	args := map[string]interface{}{
		"image_path": "/test.png",
	}
	err := ValidateRequired(args, []string{"image_path"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = ValidateRequired(args, []string{"image_path", "missing_field"})
	if err == nil {
		t.Fatal("expected error for missing field")
	}
}
