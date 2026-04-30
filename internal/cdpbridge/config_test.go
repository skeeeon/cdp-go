package cdpbridge

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeYAML(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
	return path
}

func TestLoadConfigDefaultsOnly(t *testing.T) {
	clearEnv(t)
	cfg, err := LoadConfig(nil)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Group != "239.255.76.67" || cfg.Port != 7667 {
		t.Errorf("CDP defaults: %+v", cfg)
	}
	if cfg.Broker.URL != "nats://localhost:4222" {
		t.Errorf("broker URL default: %q", cfg.Broker.URL)
	}
	if cfg.Broker.MaxReconnects != -1 {
		t.Errorf("MaxReconnects default: %d", cfg.Broker.MaxReconnects)
	}
	if cfg.Logger.Level != "info" {
		t.Errorf("logger level default: %q", cfg.Logger.Level)
	}
}

func TestLoadConfigYAMLOverridesDefaults(t *testing.T) {
	clearEnv(t)
	path := writeYAML(t, `
group: 239.1.2.3
port: 9000
prefix: foo
logger:
  level: debug
broker:
  url: nats://nats.example.com:4222
  name: alt-bridge
  max_reconnects: 5
  reconnect_wait: 3s
`)

	cfg, err := LoadConfig([]string{"--config", path})
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Group != "239.1.2.3" || cfg.Port != 9000 || cfg.Prefix != "foo" {
		t.Errorf("yaml CDP: %+v", cfg)
	}
	if cfg.Logger.Level != "debug" {
		t.Errorf("yaml logger.level: %q", cfg.Logger.Level)
	}
	if cfg.Broker.URL != "nats://nats.example.com:4222" {
		t.Errorf("yaml broker.url: %q", cfg.Broker.URL)
	}
	if cfg.Broker.Name != "alt-bridge" {
		t.Errorf("yaml broker.name: %q", cfg.Broker.Name)
	}
	if cfg.Broker.MaxReconnects != 5 {
		t.Errorf("yaml broker.max_reconnects: %d", cfg.Broker.MaxReconnects)
	}
	if cfg.Broker.ReconnectWait != 3*time.Second {
		t.Errorf("yaml broker.reconnect_wait: %s", cfg.Broker.ReconnectWait)
	}
}

func TestLoadConfigEnvOverridesYAML(t *testing.T) {
	clearEnv(t)
	path := writeYAML(t, `
group: 239.1.2.3
broker:
  url: nats://nats.yaml:4222
`)
	t.Setenv("CDP_GROUP", "239.4.5.6")
	t.Setenv("NATS_URL", "nats://nats.env:4222")

	cfg, err := LoadConfig([]string{"--config", path})
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Group != "239.4.5.6" {
		t.Errorf("env should override yaml group: got %q", cfg.Group)
	}
	if cfg.Broker.URL != "nats://nats.env:4222" {
		t.Errorf("env should override yaml broker.url: got %q", cfg.Broker.URL)
	}
}

func TestLoadConfigFlagsOverrideEnv(t *testing.T) {
	clearEnv(t)
	t.Setenv("CDP_GROUP", "239.4.5.6")
	t.Setenv("NATS_URL", "nats://nats.env:4222")

	cfg, err := LoadConfig([]string{
		"--group", "239.7.8.9",
		"--nats-url", "nats://nats.flag:4222",
	})
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Group != "239.7.8.9" {
		t.Errorf("flag should override env group: got %q", cfg.Group)
	}
	if cfg.Broker.URL != "nats://nats.flag:4222" {
		t.Errorf("flag should override env broker.url: got %q", cfg.Broker.URL)
	}
}

func TestLoadConfigUnknownYAMLKeyErrors(t *testing.T) {
	clearEnv(t)
	path := writeYAML(t, `
nonsense_key: 1
`)
	if _, err := LoadConfig([]string{"--config", path}); err == nil {
		t.Fatal("expected error for unknown yaml key, got nil")
	}
}

func TestFindConfigPathFromEqualsForm(t *testing.T) {
	clearEnv(t)
	if got := findConfigPath([]string{"--config=/tmp/x.yaml"}); got != "/tmp/x.yaml" {
		t.Errorf("--config=...: got %q", got)
	}
	if got := findConfigPath([]string{"-config=/tmp/y.yaml"}); got != "/tmp/y.yaml" {
		t.Errorf("-config=...: got %q", got)
	}
}

func TestFindConfigPathFromConfigFileEnv(t *testing.T) {
	clearEnv(t)
	t.Setenv("CONFIG_FILE", "/tmp/from-env.yaml")
	if got := findConfigPath(nil); got != "/tmp/from-env.yaml" {
		t.Errorf("CONFIG_FILE: got %q", got)
	}
}

// clearEnv removes any env var the loader looks at so individual tests
// start from a clean slate.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"CDP_GROUP", "CDP_PORT", "CDP_INTERFACE", "CDP_NATS_PREFIX",
		"CDP_UDP_READ_BUFFER",
		"LOG_LEVEL", "CONFIG_FILE",
		"GEOFENCE_PREFIX", "GEOFENCE_HYSTERESIS", "GEOFENCE_TAG_TTL",
		"NATS_URL", "NATS_NAME", "NATS_USER", "NATS_PASSWORD", "NATS_TOKEN",
		"NATS_CREDS_FILE", "NATS_NKEY_SEED_FILE",
		"NATS_TLS_CA", "NATS_TLS_CERT", "NATS_TLS_KEY", "NATS_TLS_INSECURE",
		"NATS_MAX_RECONNECTS", "NATS_RECONNECT_WAIT", "NATS_RECONNECT_JITTER",
		"NATS_PING_INTERVAL", "NATS_MAX_PINGS_OUT",
		"NATS_FLUSH_TIMEOUT", "NATS_NO_ECHO",
	} {
		if _, ok := os.LookupEnv(k); ok {
			t.Setenv(k, "")
			os.Unsetenv(k)
		}
	}
}

func TestLoadConfigGeofenceDefaultsDisabled(t *testing.T) {
	clearEnv(t)
	cfg, err := LoadConfig(nil)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Geofence.Enabled() {
		t.Error("geofence should be disabled by default (no zones configured)")
	}
	if cfg.Geofence.Prefix != "geofence" {
		t.Errorf("geofence.prefix default: %q", cfg.Geofence.Prefix)
	}
	if cfg.Geofence.Hysteresis != 5 {
		t.Errorf("geofence.hysteresis default: %d", cfg.Geofence.Hysteresis)
	}
	if cfg.Geofence.TagTTL != time.Hour {
		t.Errorf("geofence.tag_ttl default: %s", cfg.Geofence.TagTTL)
	}
	if cfg.UDPReadBuffer != 1<<20 {
		t.Errorf("udp_read_buffer default: %d", cfg.UDPReadBuffer)
	}
}

func TestLoadConfigGeofenceFromYAML(t *testing.T) {
	clearEnv(t)
	path := writeYAML(t, `
geofence:
  hysteresis: 7
  prefix: gf
  zones:
    - name: Paper
      vertices: [[0, 0], [1000, 0], [1000, 1000], [0, 1000]]
      rgb: [255, 255, 0]
    - name: Shipping
      vertices: [[2000, 0], [3000, 0], [2500, 1000]]
      rgb: [0, 0, 255]
`)
	cfg, err := LoadConfig([]string{"--config", path})
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !cfg.Geofence.Enabled() {
		t.Fatal("expected geofence enabled")
	}
	if cfg.Geofence.Hysteresis != 7 {
		t.Errorf("hysteresis: got %d, want 7", cfg.Geofence.Hysteresis)
	}
	if cfg.Geofence.Prefix != "gf" {
		t.Errorf("prefix: got %q, want gf", cfg.Geofence.Prefix)
	}
	if len(cfg.Geofence.Zones) != 2 {
		t.Fatalf("zones: got %d, want 2", len(cfg.Geofence.Zones))
	}
	zones, err := cfg.Geofence.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if zones[0].Name != "Paper" || zones[1].Name != "Shipping" {
		t.Errorf("zone names: got %q, %q", zones[0].Name, zones[1].Name)
	}
}

func TestValidateSubjectPrefix(t *testing.T) {
	good := []string{"", "cdp", "cdp.subdomain", "a.b.c"}
	for _, p := range good {
		if err := validateSubjectPrefix("test", p); err != nil {
			t.Errorf("expected %q to pass, got: %v", p, err)
		}
	}
	bad := []string{
		"cdp.*.foo",
		"cdp.>.foo",
		"cdp foo",
		"cdp\tfoo",
		"cdp.",
		".cdp",
		"cdp..foo",
	}
	for _, p := range bad {
		if err := validateSubjectPrefix("test", p); err == nil {
			t.Errorf("expected %q to fail validation, got nil", p)
		}
	}
}

func TestLoadConfigRejectsInvalidPrefix(t *testing.T) {
	clearEnv(t)
	if _, err := LoadConfig([]string{"--prefix", "cdp.*.foo"}); err == nil {
		t.Error("expected --prefix with wildcard to fail")
	}
	if _, err := LoadConfig([]string{"--geofence-prefix", "geo .bad"}); err == nil {
		t.Error("expected --geofence-prefix with whitespace to fail")
	}
}

func TestLoadConfigGeofenceFlagsOverride(t *testing.T) {
	clearEnv(t)
	path := writeYAML(t, `
geofence:
  hysteresis: 7
  prefix: yamlpfx
`)
	cfg, err := LoadConfig([]string{
		"--config", path,
		"--geofence-prefix", "flagpfx",
		"--geofence-hysteresis", "10",
	})
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Geofence.Prefix != "flagpfx" {
		t.Errorf("prefix: got %q, want flagpfx", cfg.Geofence.Prefix)
	}
	if cfg.Geofence.Hysteresis != 10 {
		t.Errorf("hysteresis: got %d, want 10", cfg.Geofence.Hysteresis)
	}
}
