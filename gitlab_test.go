package main

import (
	"reflect"
	"testing"
)

var tagTests = []struct {
	in  string
	out []project
}{
	{"projects-example.yaml",
		[]project{
			{Name: "awesomeProject",
				ID:    42,
				Token: "abcTOKENabcDEfghiV",
				Dependencies: []dependency{{
					GroupID:    "com.group",
					ArtifactID: "some-project",
				}},
			},
		},
	},
}

func TestParseClientIP(t *testing.T) {
	for _, tt := range tagTests {
		p, err := loadProjects(tt.in)
		if err != nil {
			t.Errorf("failed to load projects: %v", err)
		}

		if reflect.DeepEqual(p, tt.out) {
			t.Errorf("loadProjects(%v) => %q, want %q", tt.in, p, tt.out)
		}
	}
}
