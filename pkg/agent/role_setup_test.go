package agent

import "testing"

func TestRewriteDockerURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "localhost with port",
			in:   "http://localhost:9374/mcp/sse",
			want: "http://host.docker.internal:9374/mcp/sse",
		},
		{
			name: "127.0.0.1 with port",
			in:   "http://127.0.0.1:9374/mcp/sse",
			want: "http://host.docker.internal:9374/mcp/sse",
		},
		{
			name: "already remote host",
			in:   "http://myserver.example.com:9374/mcp/sse",
			want: "http://myserver.example.com:9374/mcp/sse",
		},
		{
			name: "empty string",
			in:   "",
			want: "",
		},
		{
			name: "localhost no port",
			in:   "http://localhost/sse",
			want: "http://host.docker.internal/sse",
		},
		{
			name: "https localhost",
			in:   "https://localhost:8443/mcp",
			want: "https://host.docker.internal:8443/mcp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rewriteDockerURL(tt.in)
			if got != tt.want {
				t.Errorf("rewriteDockerURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
