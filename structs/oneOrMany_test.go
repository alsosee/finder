package structs

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOneOrMany_UnmarshalYAML(t *testing.T) {
	type fields struct {
		Field oneOrMany `yaml:"field"`
	}

	tests := []struct {
		name string
		yaml string
		want []string
	}{
		{
			name: "string",
			yaml: "field: value\n",
			want: []string{"value"},
		},
		{
			name: "list",
			yaml: "field:\n  - value1\n  - value2\n",
			want: []string{"value1", "value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if err := yaml.Unmarshal([]byte(tt.yaml), &f); err != nil {
				t.Errorf("Unmarshal() error = %v", err)
				return
			}
			if len(f.Field) != len(tt.want) {
				t.Errorf("Unmarshal() = %v, want %v", f.Field, tt.want)
			}
			for i := range f.Field {
				if f.Field[i] != tt.want[i] {
					t.Errorf("Unmarshal() = %v, want %v", f.Field, tt.want)
				}
			}
		})
	}
}

func TestOneOrMany_MarshalYAML(t *testing.T) {
	type fields struct {
		Field oneOrMany `yaml:"field"`
	}

	tests := []struct {
		name  string
		value oneOrMany
		want  string
	}{
		{
			name:  "string",
			value: oneOrMany{"value"},
			want:  "field: value\n",
		},
		{
			name:  "list",
			value: oneOrMany{"value1", "value2"},
			want:  "field:\n    - value1\n    - value2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{Field: tt.value}
			got, err := yaml.Marshal(f)
			if err != nil {
				t.Errorf("Marshal() error = %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("Marshal() = %v, want %v", string(got), tt.want)
			}
		})
	}
}
