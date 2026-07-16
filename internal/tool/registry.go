package tool

import (
	"context"
	"fmt"
	"sort"
)

type Tool interface {
	Name() string
	Description() string
	InputSchema() map[string]interface{}
	Execute(ctx context.Context, args map[string]interface{}) (string, error)
}

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) List() []Tool {
	result := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name() < result[j].Name()
	})
	return result
}

func SanitizeArgs(args map[string]interface{}) map[string]interface{} {
	clean := make(map[string]interface{}, len(args))
	for k, v := range args {
		if k == "_meta" {
			continue
		}
		clean[k] = v
	}
	return clean
}

func ValidateRequired(args map[string]interface{}, required []string) error {
	for _, key := range required {
		if _, ok := args[key]; !ok {
			return fmt.Errorf("[参数错误] 必填参数 %s 缺失。", key)
		}
	}
	return nil
}
