package docker

import (
	"testing"
)

// ============================================
// Tests for parseEnvVar
// ============================================

func TestParseEnvVar(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantKey   string
		wantValue string
	}{
		{"simple key=value", "DB_HOST=localhost", "DB_HOST", "localhost"},
		{"empty value", "API_KEY=", "API_KEY", ""},
		{"no equals sign", "JUST_A_KEY", "JUST_A_KEY", ""},
		{"value with equals", "CONN=mysql://user:pass@host:3306/db", "CONN", "mysql://user:pass@host:3306/db"},
		{"empty string", "", "", ""},
		{"key with numbers", "PORT_8080=8080", "PORT_8080", "8080"},
		{"value with spaces", "MESSAGE=hello world", "MESSAGE", "hello world"},
		{"multiple equals in value", "URL=http://example.com/path?q=1", "URL", "http://example.com/path?q=1"},
		{"single character key", "A=apple", "A", "apple"},
		{"special chars in value", "PASSWORD=p@ssw0rd!#", "PASSWORD", "p@ssw0rd!#"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKey, gotValue := parseEnvVar(tt.input)
			if gotKey != tt.wantKey {
				t.Errorf("parseEnvVar(%q) key = %q, want %q", tt.input, gotKey, tt.wantKey)
			}
			if gotValue != tt.wantValue {
				t.Errorf("parseEnvVar(%q) value = %q, want %q", tt.input, gotValue, tt.wantValue)
			}
		})
	}
}

// ============================================
// Tests for splitCommand
// ============================================

func TestSplitCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty string", "", nil},
		{"single word", "nginx", []string{"nginx"}},
		{"simple command", "nginx -g daemon off", []string{"nginx", "-g", "daemon", "off"}},
		{"command with single quotes", `nginx -g 'daemon off;'`, []string{"nginx", "-g", "daemon off;"}},
		{"command with double quotes", `nginx -g "daemon off;"`, []string{"nginx", "-g", "daemon off;"}},
		// Note: splitCommand uses a simple toggle for both quote types,
		// so mixed/nested quotes cause the inner quote to toggle quoting off.
		// This is a known limitation of the simple implementation.
		{"mixed quotes", `sh -c "echo 'hello world'"`, []string{"sh", "-c", "echo hello", "world"}},
		{"quoted at start", `"long arg" -x`, []string{"long arg", "-x"}},
		{"multiple quoted strings", `--name "my container" --port 8080`, []string{"--name", "my container", "--port", "8080"}},
		{"trailing quoted string", `echo "hello world"`, []string{"echo", "hello world"}},
		{"empty quoted string", `--name ""`, []string{"--name"}},
		// Single quotes inside double quotes: ' toggles quoting off, splitting the string
		{"single quotes inside double quotes", `cmd "it's working"`, []string{"cmd", "its", "working"}},
		{"multiple spaces", "cmd   arg1    arg2", []string{"cmd", "arg1", "arg2"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitCommand(tt.input)

			// Check length
			if len(result) != len(tt.expected) {
				t.Errorf("splitCommand(%q) returned %d parts: %v, want %d parts: %v",
					tt.input, len(result), result, len(tt.expected), tt.expected)
				return
			}

			// Check each element
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("splitCommand(%q)[%d] = %q, want %q", tt.input, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

// ============================================
// Tests for ParseComposeFile
// ============================================

func TestParseComposeFile(t *testing.T) {
	t.Run("valid compose file with one service", func(t *testing.T) {
		yaml := `
version: '3'
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
    environment:
      - NGINX_HOST=localhost
    volumes:
      - ./html:/usr/share/nginx/html
    networks:
      - frontend
    restart: always
    labels:
      app: web
`

		compose, err := ParseComposeFile(yaml)
		if err != nil {
			t.Fatalf("ParseComposeFile returned unexpected error: %v", err)
		}

		if compose.Version != "3" {
			t.Errorf("Version = %q, want %q", compose.Version, "3")
		}

		if len(compose.Services) != 1 {
			t.Fatalf("len(Services) = %d, want 1", len(compose.Services))
		}

		svc, ok := compose.Services["web"]
		if !ok {
			t.Fatal("expected service 'web' not found")
		}

		if svc.Image != "nginx:latest" {
			t.Errorf("Image = %q, want %q", svc.Image, "nginx:latest")
		}
		if len(svc.Ports) != 1 || svc.Ports[0] != "8080:80" {
			t.Errorf("Ports = %v, want [8080:80]", svc.Ports)
		}
		if len(svc.Environment) != 1 || svc.Environment[0] != "NGINX_HOST=localhost" {
			t.Errorf("Environment = %v, want [NGINX_HOST=localhost]", svc.Environment)
		}
		if len(svc.Volumes) != 1 || svc.Volumes[0] != "./html:/usr/share/nginx/html" {
			t.Errorf("Volumes = %v, want [./html:/usr/share/nginx/html]", svc.Volumes)
		}
		if len(svc.Networks) != 1 || svc.Networks[0] != "frontend" {
			t.Errorf("Networks = %v, want [frontend]", svc.Networks)
		}
		if svc.Restart != "always" {
			t.Errorf("Restart = %q, want %q", svc.Restart, "always")
		}
		if svc.Labels["app"] != "web" {
			t.Errorf("Labels[app] = %q, want %q", svc.Labels["app"], "web")
		}
	})

	t.Run("valid compose file with multiple services and depends_on", func(t *testing.T) {
		// Environment must use YAML list format with "-" prefix for []string field
		yaml := `
version: '3.8'
services:
  db:
    image: postgres:16
    environment:
      - POSTGRES_PASSWORD=secret
      - POSTGRES_DB=myapp
  web:
    image: myapp:latest
    ports:
      - "3000:3000"
    depends_on:
      - db
  redis:
    image: redis:7-alpine
`
		compose, err := ParseComposeFile(yaml)
		if err != nil {
			t.Fatalf("ParseComposeFile returned unexpected error: %v", err)
		}

		if len(compose.Services) != 3 {
			t.Fatalf("len(Services) = %d, want 3", len(compose.Services))
		}

		// Check db service
		dbSvc := compose.Services["db"]
		if dbSvc.Image != "postgres:16" {
			t.Errorf("db.Image = %q, want %q", dbSvc.Image, "postgres:16")
		}
		if len(dbSvc.Environment) != 2 {
			t.Errorf("db.Environment count = %d, want 2", len(dbSvc.Environment))
		}

		// Check web service depends_on
		webSvc := compose.Services["web"]
		if len(webSvc.DependsOn) != 1 || webSvc.DependsOn[0] != "db" {
			t.Errorf("web.DependsOn = %v, want [db]", webSvc.DependsOn)
		}

		// Check redis service
		redisSvc := compose.Services["redis"]
		if redisSvc.Image != "redis:7-alpine" {
			t.Errorf("redis.Image = %q, want %q", redisSvc.Image, "redis:7-alpine")
		}
	})

	t.Run("compose file with networks and volumes", func(t *testing.T) {
		yaml := `
version: '3'
services:
  app:
    image: myapp:latest
    networks:
      - frontend
      - backend
networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge
volumes:
  data:
`
		compose, err := ParseComposeFile(yaml)
		if err != nil {
			t.Fatalf("ParseComposeFile returned unexpected error: %v", err)
		}

		// Check networks
		if len(compose.Networks) != 2 {
			t.Fatalf("len(Networks) = %d, want 2", len(compose.Networks))
		}
		if _, ok := compose.Networks["frontend"]; !ok {
			t.Error("expected network 'frontend' not found")
		}
		if _, ok := compose.Networks["backend"]; !ok {
			t.Error("expected network 'backend' not found")
		}

		// Check volumes
		if len(compose.Volumes) != 1 {
			t.Fatalf("len(Volumes) = %d, want 1", len(compose.Volumes))
		}
		if _, ok := compose.Volumes["data"]; !ok {
			t.Error("expected volume 'data' not found")
		}

		// Check service networks
		appSvc := compose.Services["app"]
		if len(appSvc.Networks) != 2 {
			t.Fatalf("app.Networks count = %d, want 2", len(appSvc.Networks))
		}
	})

	t.Run("valid compose with no version specified", func(t *testing.T) {
		yaml := `
services:
  worker:
    image: alpine:latest
    command: sleep infinity
`
		compose, err := ParseComposeFile(yaml)
		if err != nil {
			t.Fatalf("ParseComposeFile returned unexpected error: %v", err)
		}

		if compose.Version != "" {
			t.Errorf("Version should be empty when not specified, got %q", compose.Version)
		}

		if len(compose.Services) != 1 {
			t.Fatalf("len(Services) = %d, want 1", len(compose.Services))
		}

		workerSvc := compose.Services["worker"]
		if workerSvc.Command != "sleep infinity" {
			t.Errorf("Command = %q, want %q", workerSvc.Command, "sleep infinity")
		}
	})

	t.Run("parse compose file with command in string format", func(t *testing.T) {
		yaml := `
services:
  app:
    image: node:20
    command: node server.js
`
		compose, err := ParseComposeFile(yaml)
		if err != nil {
			t.Fatalf("ParseComposeFile returned unexpected error: %v", err)
		}

		appSvc := compose.Services["app"]
		if appSvc.Command != "node server.js" {
			t.Errorf("Command = %q, want %q", appSvc.Command, "node server.js")
		}
	})

	t.Run("compose file with UDP port", func(t *testing.T) {
		yaml := `
services:
  dns:
    image: coredns:latest
    ports:
      - "53:53/udp"
      - "53:53/tcp"
`
		compose, err := ParseComposeFile(yaml)
		if err != nil {
			t.Fatalf("ParseComposeFile returned unexpected error: %v", err)
		}

		dnsSvc := compose.Services["dns"]
		if len(dnsSvc.Ports) != 2 {
			t.Fatalf("len(Ports) = %d, want 2", len(dnsSvc.Ports))
		}
		if dnsSvc.Ports[0] != "53:53/udp" {
			t.Errorf("Port[0] = %q, want %q", dnsSvc.Ports[0], "53:53/udp")
		}
		if dnsSvc.Ports[1] != "53:53/tcp" {
			t.Errorf("Port[1] = %q, want %q", dnsSvc.Ports[1], "53:53/tcp")
		}
	})

	t.Run("invalid YAML returns error", func(t *testing.T) {
		yaml := `invalid: yaml: [content`
		_, err := ParseComposeFile(yaml)
		if err == nil {
			t.Fatal("expected error for invalid YAML, got nil")
		}
	})

	t.Run("empty YAML returns empty services", func(t *testing.T) {
		yaml := ``
		compose, err := ParseComposeFile(yaml)
		if err != nil {
			t.Fatalf("ParseComposeFile returned unexpected error: %v", err)
		}
		if compose.Services != nil {
			t.Errorf("Services should be nil for empty YAML, got %v", compose.Services)
		}
	})

	t.Run("services with no services key", func(t *testing.T) {
		yaml := `version: '3'`
		compose, err := ParseComposeFile(yaml)
		if err != nil {
			t.Fatalf("ParseComposeFile returned unexpected error: %v", err)
		}
		if len(compose.Services) != 0 {
			t.Errorf("len(Services) = %d, want 0", len(compose.Services))
		}
	})
}

// ============================================
// Integration test: splitCommand + parseEnvVar used together
// ============================================

func TestParseComposeServiceRequest(t *testing.T) {
	// Simulate the request flow: parse compose YAML, then verify helper functions
	t.Run("parse compose and verify environment parsing", func(t *testing.T) {
		// Environment must use YAML list format with "-" prefix for []string field
		yaml := `
services:
  api:
    image: myapi:latest
    environment:
      - DB_URL=postgres://localhost:5432/mydb
      - DEBUG=true
      - EMPTY=
`
		compose, err := ParseComposeFile(yaml)
		if err != nil {
			t.Fatalf("ParseComposeFile error: %v", err)
		}

		svc := compose.Services["api"]

		// Test parseEnvVar for each env var
		for _, env := range svc.Environment {
			key, val := parseEnvVar(env)
			switch key {
			case "DB_URL":
				if val != "postgres://localhost:5432/mydb" {
					t.Errorf("DB_URL = %q, want %q", val, "postgres://localhost:5432/mydb")
				}
			case "DEBUG":
				if val != "true" {
					t.Errorf("DEBUG = %q, want %q", val, "true")
				}
			case "EMPTY":
				if val != "" {
					t.Errorf("EMPTY = %q, want empty string", val)
				}
			default:
				t.Errorf("unexpected env var key: %q", key)
			}
		}
	})

	t.Run("parse compose and verify command splitting", func(t *testing.T) {
		// Use single quotes for the shell command so splitCommand handles it properly
		yaml := `
services:
  worker:
    image: alpine:latest
    command: 'sh -c "while true; do echo working; sleep 5; done"'
`
		compose, err := ParseComposeFile(yaml)
		if err != nil {
			t.Fatalf("ParseComposeFile error: %v", err)
		}

		svc := compose.Services["worker"]
		parts := splitCommand(svc.Command)

		// splitCommand handles this as: sh, -c, "while true; do echo working; sleep 5; done"
		// since the single quotes wrap the outer string and double quotes are inside
		if len(parts) != 3 {
			t.Fatalf("splitCommand returned %d parts: %v, want 3", len(parts), parts)
		}

		expected := []string{"sh", "-c", "while true; do echo working; sleep 5; done"}
		for i := range parts {
			if parts[i] != expected[i] {
				t.Errorf("parts[%d] = %q, want %q", i, parts[i], expected[i])
			}
		}
	})
}
