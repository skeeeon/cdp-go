// Package config holds shared configuration types and a tiny loader used
// by every binary in this repo.
//
// The loader applies values in this priority order (lowest to highest):
//
//  1. Built-in defaults (whatever the caller seeds the struct with)
//  2. YAML config file at the path given by --config / CONFIG_FILE
//  3. Environment variables
//  4. Command-line flags
//
// Each binary defines its own top-level struct (e.g. cdpbridge.Config) and
// embeds the shared Broker and Logger structs from this package. The loader
// is generic over that struct via a yaml.Unmarshal pass plus a flag set
// the caller registers.
package config

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Broker holds the NATS connection knobs every binary needs. yaml tags
// drive YAML decoding; binaries map env vars and flags onto these fields
// in their own loader.
type Broker struct {
	URL              string        `yaml:"url"`
	Name             string        `yaml:"name"`
	User             string        `yaml:"user"`
	Password         string        `yaml:"password"`
	Token            string        `yaml:"token"`
	CredsFile        string        `yaml:"creds_file"`
	NkeySeedFile     string        `yaml:"nkey_seed_file"`
	TLSCA            string        `yaml:"tls_ca"`
	TLSCert          string        `yaml:"tls_cert"`
	TLSKey           string        `yaml:"tls_key"`
	TLSInsecure      bool          `yaml:"tls_insecure"`
	MaxReconnects    int           `yaml:"max_reconnects"`
	ReconnectWait    time.Duration `yaml:"reconnect_wait"`
	ReconnectJitter  time.Duration `yaml:"reconnect_jitter"`
	PingInterval     time.Duration `yaml:"ping_interval"`
	MaxPingsOut      int           `yaml:"max_pings_out"`
	DrainTimeout     time.Duration `yaml:"drain_timeout"`
	FlushTimeout     time.Duration `yaml:"flush_timeout"`
	NoEcho           bool          `yaml:"no_echo"`
}

// Logger holds slog setup knobs.
type Logger struct {
	Level string `yaml:"level"` // debug | info | warn | error
}

// LoadYAMLInto reads path (if non-empty) and unmarshals it into out.
//
// A missing path is not an error: the function returns nil so callers can
// pass an optional --config flag and skip reading when unset.
//
// out must be a pointer to a struct with appropriate yaml tags.
func LoadYAMLInto(path string, out any) error {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("config: read %s: %w", path, err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true) // surface typos in keys
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("config: parse %s: %w", path, err)
	}
	return nil
}
