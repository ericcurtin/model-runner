package huggingface

import (
	"testing"
)

func TestClassifyFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     fileType
	}{
		{"safetensors file", "model.safetensors", fileTypeSafetensors},
		{"safetensors uppercase", "model.SAFETENSORS", fileTypeSafetensors},
		{"safetensors mixed case", "Model.SafeTensors", fileTypeSafetensors},
		{"sharded safetensors", "model-00001-of-00003.safetensors", fileTypeSafetensors},

		{"json config", "config.json", fileTypeConfig},
		{"tokenizer json", "tokenizer.json", fileTypeConfig},
		{"tokenizer config", "tokenizer_config.json", fileTypeConfig},
		{"txt file", "README.txt", fileTypeConfig},
		{"markdown file", "README.md", fileTypeConfig},
		{"vocab file", "vocab.vocab", fileTypeConfig},
		{"jinja template", "chat_template.jinja", fileTypeConfig},
		{"tokenizer model", "tokenizer.model", fileTypeConfig},

		{"unknown extension", "model.bin", fileTypeUnknown},
		{"python file", "model.py", fileTypeUnknown},
		{"pytorch model", "pytorch_model.bin", fileTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyFile(tt.filename); got != tt.want {
				t.Errorf("classifyFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestFilterModelFiles(t *testing.T) {
	files := []RepoFile{
		{Type: "file", Path: "model.safetensors", Size: 1000},
		{Type: "file", Path: "config.json", Size: 100},
		{Type: "file", Path: "tokenizer.json", Size: 200},
		{Type: "file", Path: "README.md", Size: 50},
		{Type: "file", Path: "model.py", Size: 500},
		{Type: "directory", Path: "subdir", Size: 0},
		{Type: "file", Path: "model-00001-of-00002.safetensors", Size: 2000},
		{Type: "file", Path: "model-00002-of-00002.safetensors", Size: 2000},
	}

	safetensors, configs := FilterModelFiles(files)

	if len(safetensors) != 3 {
		t.Errorf("Expected 3 safetensors files, got %d", len(safetensors))
	}
	if len(configs) != 3 {
		t.Errorf("Expected 3 config files, got %d", len(configs))
	}
}

func TestTotalSize(t *testing.T) {
	files := []RepoFile{
		{Type: "file", Path: "a.safetensors", Size: 1000},
		{Type: "file", Path: "b.safetensors", Size: 2000, LFS: &LFSInfo{Size: 5000}},
	}

	total := TotalSize(files)
	if total != 6000 { // 1000 + 5000 (LFS size takes precedence)
		t.Errorf("TotalSize() = %d, want 6000", total)
	}
}

func TestRepoFileActualSize(t *testing.T) {
	tests := []struct {
		name string
		file RepoFile
		want int64
	}{
		{
			name: "regular file",
			file: RepoFile{Size: 1000},
			want: 1000,
		},
		{
			name: "LFS file",
			file: RepoFile{Size: 100, LFS: &LFSInfo{Size: 5000}},
			want: 5000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.file.ActualSize(); got != tt.want {
				t.Errorf("ActualSize() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestIsSafetensorsModel(t *testing.T) {
	tests := []struct {
		name  string
		files []RepoFile
		want  bool
	}{
		{
			name: "has safetensors",
			files: []RepoFile{
				{Type: "file", Path: "model.safetensors"},
				{Type: "file", Path: "config.json"},
			},
			want: true,
		},
		{
			name: "no safetensors",
			files: []RepoFile{
				{Type: "file", Path: "config.json"},
				{Type: "file", Path: "README.md"},
			},
			want: false,
		},
		{
			name:  "empty",
			files: []RepoFile{},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSafetensorsModel(tt.files); got != tt.want {
				t.Errorf("isSafetensorsModel() = %v, want %v", got, tt.want)
			}
		})
	}
}
