package proxy

import (
	"testing"
)

func TestIsHidden_EmptyFilter(t *testing.T) {
	p := &Proxy{config: &Config{Filter: nil}}
	if p.isHidden("any_tool") {
		t.Fatal("empty filter should not hide any tool")
	}
}

func TestIsHidden_ExactMatch(t *testing.T) {
	p := &Proxy{config: &Config{Filter: []string{"file_read"}}}
	if !p.isHidden("file_read") {
		t.Fatal("exact match should be hidden")
	}
}

func TestIsHidden_CaseSensitive(t *testing.T) {
	p := &Proxy{config: &Config{Filter: []string{"file_read"}}}
	if p.isHidden("File_Read") {
		t.Fatal("filter should be case-sensitive")
	}
}

func TestIsHidden_MultipleFilters(t *testing.T) {
	p := &Proxy{config: &Config{Filter: []string{"file_read", "file_write", "web_fetch"}}}
	if !p.isHidden("file_write") {
		t.Fatal("file_write should be hidden")
	}
	if p.isHidden("file_list") {
		t.Fatal("file_list should not be hidden")
	}
}

func TestIsHidden_TraceToolsNotSpecial(t *testing.T) {
	p := &Proxy{config: &Config{Filter: []string{"trace.history"}}}
	if !p.isHidden("trace.history") {
		t.Fatal("trace.history should be hidden when in filter")
	}
}
