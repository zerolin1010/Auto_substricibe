package configs

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// 设置测试环境变量
	os.Setenv("JELLY_URL", "https://test.com")
	os.Setenv("JELLY_API_KEY", "test-key")
	os.Setenv("MP_URL", "http://test.com")
	os.Setenv("MP_USERNAME", "test-user")
	os.Setenv("MP_PASSWORD", "test-pass")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.JellyURL != "https://test.com" {
		t.Errorf("JellyURL = %v, want %v", cfg.JellyURL, "https://test.com")
	}

	if cfg.MPURL != "http://test.com" {
		t.Errorf("MPURL = %v, want %v", cfg.MPURL, "http://test.com")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				JellyURL:        "https://test.com",
				JellyAPIKey:     "key",
				MPURL:           "http://test.com",
				MPUsername:      "user",
				MPPassword:      "pass",
				MPAuthScheme:    "bearer",
				MPTVEpisodeMode: "season",
				StoreType:       "sqlite",
			},
			wantErr: false,
		},
		{
			name: "missing JELLY_URL",
			cfg: &Config{
				JellyAPIKey: "key",
				MPURL:       "http://test.com",
				MPUsername:  "user",
				MPPassword:  "pass",
			},
			wantErr: true,
		},
		{
			name: "invalid auth scheme",
			cfg: &Config{
				JellyURL:     "https://test.com",
				JellyAPIKey:  "key",
				MPURL:        "http://test.com",
				MPUsername:   "user",
				MPPassword:   "pass",
				MPAuthScheme: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMaskSensitive(t *testing.T) {
	cfg := &Config{
		JellyAPIKey: "1234567890abcdef",
		MPPassword:  "abcdefghijklmnop",
	}

	masked := cfg.MaskSensitive()

	if masked["jelly_api_key"] == cfg.JellyAPIKey {
		t.Error("JellyAPIKey should be masked")
	}

	if masked["mp_password"] == cfg.MPPassword {
		t.Error("MPPassword should be masked")
	}
}
