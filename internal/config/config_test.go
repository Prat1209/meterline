package config

import "testing"

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		wantErr  bool
		wantPort int
	}{
		{"defaults port", map[string]string{"DATABASE_URL": "postgres://x"}, false, 8080},
		{"custom port", map[string]string{"DATABASE_URL": "postgres://x", "PORT": "9000"}, false, 9000},
		{"missing database url", map[string]string{}, true, 0},
		{"bad port", map[string]string{"DATABASE_URL": "postgres://x", "PORT": "abc"}, true, 0},
		{"test key ok", map[string]string{"DATABASE_URL": "postgres://x", "STRIPE_SECRET_KEY": "sk_test_123"}, false, 8080},
		{"live key rejected", map[string]string{"DATABASE_URL": "postgres://x", "STRIPE_SECRET_KEY": "sk_live_123"}, true, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := load(env(tc.env))
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && cfg.Port != tc.wantPort {
				t.Errorf("Port = %d, want %d", cfg.Port, tc.wantPort)
			}
		})
	}
}
