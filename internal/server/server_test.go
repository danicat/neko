package server

import (
	"testing"

	"github.com/danicat/neko/internal/backend"
	"github.com/danicat/neko/internal/core/config"
)

func TestServer_RegisterHandlers_DisableTools(t *testing.T) {
	tests := []struct {
		name          string
		disabledTools map[string]bool
		wantErr       bool
	}{
		{
			name:          "no disabled tools",
			disabledTools: map[string]bool{},
			wantErr:       false,
		},
		{
			name:          "disable build",
			disabledTools: map[string]bool{"build": true},
			wantErr:       false,
		},
		{
			name:          "disable create_file",
			disabledTools: map[string]bool{"create_file": true},
			wantErr:       false,
		},
		{
			name:          "disable review_code",
			disabledTools: map[string]bool{"review_code": true},
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				DisabledTools: tt.disabledTools,
			}
			reg := backend.NewRegistry()
			s := New(cfg, "test", reg)
			err := s.RegisterHandlers()
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterHandlers() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
