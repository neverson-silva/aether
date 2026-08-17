package application

import "testing"

func TestSplitImage(t *testing.T) {
	cases := []struct {
		name, repo, tag string
	}{
		{"nginx:alpine", "nginx", "alpine"},
		{"docker.io/library/nginx:latest", "docker.io/library/nginx", "latest"},
		{"localhost:5000/app:v1", "localhost:5000/app", "v1"},
		{"untagged", "untagged", "latest"},
	}
	for _, c := range cases {
		repo, tag := splitImage(c.name)
		if repo != c.repo || tag != c.tag {
			t.Fatalf("splitImage(%q) = %q,%q want %q,%q", c.name, repo, tag, c.repo, c.tag)
		}
	}
}
