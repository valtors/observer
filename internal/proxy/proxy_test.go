package proxy

import (
	"testing"
)

func TestIsHidden(t *testing.T) {
	tests := []struct {
		name     string
		filter   []string
		toolName string
		want     bool
	}{
		{
			name:     "empty filter allows all",
			filter:   []string{},
			toolName: "any_tool",
			want:     false,
		},
		{
			name:     "exact match is hidden",
			filter:   []string{"hidden_tool"},
			toolName: "hidden_tool",
			want:     true,
		},
		{
			name:     "no match is visible",
			filter:   []string{"other_tool"},
			toolName: "visible_tool",
			want:     false,
		},
		{
			name:     "case sensitive match",
			filter:   []string{"Hidden_Tool"},
			toolName: "hidden_tool",
			want:     false,
		},
		{
			name:     "multiple items in filter",
			filter:   []string{"toolA", "toolB", "toolC"},
			toolName: "toolB",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Proxy{
				config: &Config{Filter: tt.filter},
			}
			if got := p.isHidden(tt.toolName); got != tt.want {
				t.Errorf("isHidden() = %v, want %v", got, tt.want)
			}
		})
	}
}
