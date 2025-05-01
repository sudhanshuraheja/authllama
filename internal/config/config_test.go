package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantPort string
		wantErr  bool
	}{
		{
			name: "valid config with port",
			content: `{
				"_global": {
					"port": "8000",
					"ollama_url": "http://localhost:11434",
					"admin_api_key": "key123"
				},
				"services": {
					"test": {
						"auth": "abc",
						"tpm": 10
					}
				}
			}`,
			wantPort: "8000",
			wantErr:  false,
		},
		{
			name: "missing port uses default",
			content: `{
				"_global": {
					"ollama_url": "http://localhost:11434",
					"admin_api_key": "key123"
				},
				"services": {
					"test": {
						"auth": "abc",
						"tpm": 10
					}
				}
			}`,
			wantPort: "3000",
			wantErr:  false,
		},
		{
			name: "invalid json",
			content: `{
				"_global": {
					"port": 7000,
					"ollama_url": "http://localhost:11434",
					"admin_api_key": "key123"
				},`,
			wantErr: true,
		},
		{
			name: "missing required service auth",
			content: `{
				"_global": {
					"port": "8000",
					"ollama_url": "http://localhost:11434",
					"admin_api_key": "key123"
				},
				"services": {
					"test": {
						"tpm": 10
					}
				}
			}`,
			wantPort: "8000",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(tmpfile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			cfg, err := LoadConfig(tmpfile)
			if (err != nil) != tt.wantErr {
				t.Fatalf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && cfg.Global.Port != tt.wantPort {
				t.Errorf("expected port %s, got %s", tt.wantPort, cfg.Global.Port)
			}
		})
	}
}
