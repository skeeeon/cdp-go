package cdpbridge

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/velociti/cdp-go/internal/config"
)

// Config is the cdp-nats-bridge binary's full configuration.
//
// CDP-listener fields are direct; broker and logger settings are reused
// from internal/config so a future binary can share them.
type Config struct {
	// CDP listener
	Group  string `yaml:"group"`
	Port   int    `yaml:"port"`
	Iface  string `yaml:"iface"`
	Prefix string `yaml:"prefix"`

	Logger config.Logger `yaml:"logger"`
	Broker config.Broker `yaml:"broker"`
}

// defaults seeds a Config with the same values flags would otherwise use.
// Splitting this out lets the YAML loader and the test suite share them.
func defaults() *Config {
	return &Config{
		Group:  "239.255.76.67",
		Port:   7667,
		Prefix: "cdp",
		Logger: config.Logger{Level: "info"},
		Broker: config.Broker{
			URL:             "nats://localhost:4222",
			Name:            "cdp-nats-bridge",
			MaxReconnects:   -1,
			ReconnectWait:   2 * time.Second,
			ReconnectJitter: 100 * time.Millisecond,
			PingInterval:    2 * time.Minute,
			MaxPingsOut:     2,
			DrainTimeout:    30 * time.Second,
			FlushTimeout:    5 * time.Second,
		},
	}
}

// LoadConfig resolves config in priority order: defaults → YAML → env → flags.
//
// The YAML file path comes from --config or CONFIG_FILE; if both are empty
// the file step is skipped. Env vars use the names documented in the README
// (e.g. CDP_GROUP, NATS_URL, LOG_LEVEL).
func LoadConfig(args []string) (*Config, error) {
	cfg := defaults()

	// 1. Peek for --config / CONFIG_FILE before declaring any other flags.
	configPath := findConfigPath(args)
	if err := config.LoadYAMLInto(configPath, cfg); err != nil {
		return nil, err
	}

	// 2. Apply env vars on top of file values.
	applyEnv(cfg)

	// 3. Define flags whose defaults are the now-resolved values; flag.Parse
	//    only overrides what the operator typed on the command line.
	fs := flag.NewFlagSet("cdp-nats-bridge", flag.ContinueOnError)

	// (allow --config to appear at parse time as well, even though we
	// already consumed it; otherwise flag.Parse complains)
	var configFlag string
	fs.StringVar(&configFlag, "config", configPath, "path to YAML config file")

	// CDP
	fs.StringVar(&cfg.Group, "group", cfg.Group, "CDP multicast group")
	fs.IntVar(&cfg.Port, "port", cfg.Port, "CDP UDP port")
	fs.StringVar(&cfg.Iface, "iface", cfg.Iface, "Network interface name (optional)")
	fs.StringVar(&cfg.Prefix, "prefix", cfg.Prefix, "NATS subject prefix")

	// Logger
	fs.StringVar(&cfg.Logger.Level, "log-level", cfg.Logger.Level, "log level: debug|info|warn|error")

	// Broker
	fs.StringVar(&cfg.Broker.URL, "nats-url", cfg.Broker.URL, "NATS server URL(s); comma-separated")
	fs.StringVar(&cfg.Broker.Name, "nats-name", cfg.Broker.Name, "NATS connection name")
	fs.StringVar(&cfg.Broker.User, "nats-user", cfg.Broker.User, "NATS username (paired with --nats-password)")
	fs.StringVar(&cfg.Broker.Password, "nats-password", cfg.Broker.Password, "NATS password")
	fs.StringVar(&cfg.Broker.Token, "nats-token", cfg.Broker.Token, "NATS auth token")
	fs.StringVar(&cfg.Broker.CredsFile, "nats-creds", cfg.Broker.CredsFile, "NATS user credentials file (JWT)")
	fs.StringVar(&cfg.Broker.NkeySeedFile, "nats-nkey", cfg.Broker.NkeySeedFile, "NATS NKey seed file")
	fs.StringVar(&cfg.Broker.TLSCA, "nats-tls-ca", cfg.Broker.TLSCA, "NATS TLS CA bundle path")
	fs.StringVar(&cfg.Broker.TLSCert, "nats-tls-cert", cfg.Broker.TLSCert, "NATS TLS client cert path")
	fs.StringVar(&cfg.Broker.TLSKey, "nats-tls-key", cfg.Broker.TLSKey, "NATS TLS client key path")
	fs.BoolVar(&cfg.Broker.TLSInsecure, "nats-tls-insecure", cfg.Broker.TLSInsecure, "skip NATS TLS verification (dev only)")
	fs.IntVar(&cfg.Broker.MaxReconnects, "nats-max-reconnects", cfg.Broker.MaxReconnects, "NATS max reconnect attempts (-1 = forever)")
	fs.DurationVar(&cfg.Broker.ReconnectWait, "nats-reconnect-wait", cfg.Broker.ReconnectWait, "NATS reconnect base delay")
	fs.DurationVar(&cfg.Broker.ReconnectJitter, "nats-reconnect-jitter", cfg.Broker.ReconnectJitter, "NATS reconnect jitter")
	fs.DurationVar(&cfg.Broker.PingInterval, "nats-ping-interval", cfg.Broker.PingInterval, "NATS ping interval")
	fs.IntVar(&cfg.Broker.MaxPingsOut, "nats-max-pings-out", cfg.Broker.MaxPingsOut, "NATS max outstanding pings before disconnect")
	fs.DurationVar(&cfg.Broker.DrainTimeout, "nats-drain-timeout", cfg.Broker.DrainTimeout, "NATS drain timeout on shutdown")
	fs.DurationVar(&cfg.Broker.FlushTimeout, "nats-flush-timeout", cfg.Broker.FlushTimeout, "NATS flush timeout")
	fs.BoolVar(&cfg.Broker.NoEcho, "nats-no-echo", cfg.Broker.NoEcho, "NATS NoEcho: suppress own messages")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	return cfg, nil
}

// findConfigPath walks args looking for --config / -config so we can read
// the YAML before defining any other flags. Falls back to CONFIG_FILE.
func findConfigPath(args []string) string {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--config" || a == "-config" {
			if i+1 < len(args) {
				return args[i+1]
			}
		}
		for _, prefix := range []string{"--config=", "-config="} {
			if strings.HasPrefix(a, prefix) {
				return strings.TrimPrefix(a, prefix)
			}
		}
	}
	return os.Getenv("CONFIG_FILE")
}

// applyEnv overlays known environment variables onto cfg in-place.
func applyEnv(cfg *Config) {
	envStr(&cfg.Group, "CDP_GROUP")
	envInt(&cfg.Port, "CDP_PORT")
	envStr(&cfg.Iface, "CDP_INTERFACE")
	envStr(&cfg.Prefix, "CDP_NATS_PREFIX")

	envStr(&cfg.Logger.Level, "LOG_LEVEL")

	envStr(&cfg.Broker.URL, "NATS_URL")
	envStr(&cfg.Broker.Name, "NATS_NAME")
	envStr(&cfg.Broker.User, "NATS_USER")
	envStr(&cfg.Broker.Password, "NATS_PASSWORD")
	envStr(&cfg.Broker.Token, "NATS_TOKEN")
	envStr(&cfg.Broker.CredsFile, "NATS_CREDS_FILE")
	envStr(&cfg.Broker.NkeySeedFile, "NATS_NKEY_SEED_FILE")
	envStr(&cfg.Broker.TLSCA, "NATS_TLS_CA")
	envStr(&cfg.Broker.TLSCert, "NATS_TLS_CERT")
	envStr(&cfg.Broker.TLSKey, "NATS_TLS_KEY")
	envBool(&cfg.Broker.TLSInsecure, "NATS_TLS_INSECURE")
	envInt(&cfg.Broker.MaxReconnects, "NATS_MAX_RECONNECTS")
	envDur(&cfg.Broker.ReconnectWait, "NATS_RECONNECT_WAIT")
	envDur(&cfg.Broker.ReconnectJitter, "NATS_RECONNECT_JITTER")
	envDur(&cfg.Broker.PingInterval, "NATS_PING_INTERVAL")
	envInt(&cfg.Broker.MaxPingsOut, "NATS_MAX_PINGS_OUT")
	envDur(&cfg.Broker.DrainTimeout, "NATS_DRAIN_TIMEOUT")
	envDur(&cfg.Broker.FlushTimeout, "NATS_FLUSH_TIMEOUT")
	envBool(&cfg.Broker.NoEcho, "NATS_NO_ECHO")
}

func envStr(dst *string, key string) {
	if v, ok := os.LookupEnv(key); ok {
		*dst = v
	}
}

func envInt(dst *int, key string) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %s=%q is not a valid integer; ignoring\n", key, v)
		return
	}
	*dst = n
}

func envBool(dst *bool, key string) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %s=%q is not a valid bool; ignoring\n", key, v)
		return
	}
	*dst = b
}

func envDur(dst *time.Duration, key string) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %s=%q is not a valid duration; ignoring\n", key, v)
		return
	}
	*dst = d
}
