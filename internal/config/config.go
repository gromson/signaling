package config

import (
	"flag"
	"os"

	"github.com/pkg/errors"
	"gopkg.in/ini.v1"
)

const (
	envAddr          = "STOP_PANIC_ADDR"
	envTlsCert       = "STOP_PANIC_TLS_CERT"
	envTlsKey        = "STOP_PANIC_TLS_KEY"
	envAllowedOrigin = "STOP_PANIC_ALLOWED_ORIGIN"
	envLogsLevel     = "STOP_PANIC_LOGS_LEVEL"
	envLogsFormat    = "STOP_PANIC_LOGS_FORMAT"
	envAppleCert     = "STOP_PANIC_APPLE_CERT"
	envAppleBundle   = "STOP_PANIC_APPLE_BUNDLE"
)

var (
	configFile      string
	addr            string
	tlsCert, tlsKey string
	allowedOrigin   string
	loggingLevel    string
	loggingFormat   string
)

func init() {
	flag.StringVar(&configFile, "config", "/etc/stop-panic/config.ini", "Path to config file")

	flag.StringVar(&addr, "addr", ":8080", "http service address")
	flag.StringVar(&tlsCert, "tls-cert", "", "path to tls certificate file")
	flag.StringVar(&tlsKey, "tls-key", "", "path to tls key file")
	flag.StringVar(&allowedOrigin, "allowed-origin", "*", "origin that allowed to connect to the server")
	flag.StringVar(&loggingLevel, "logging-level", "info", "logging level")
	flag.StringVar(&loggingFormat, "logging-format", "json", "logging format (options: json, text)")
	flag.Parse()
}

type Config struct {
	Server Server
	Logs   Logs
	Apple  Apple
}

type Server struct {
	Addr          string
	TlsCert       string
	TlsKey        string
	AllowedOrigin string
}

type Logs struct {
	Level  string
	Format string
}

type Apple struct {
	Cert   string
	Bundle string
}

func GetConfig() (*Config, error) {
	conf := createFromEnv()

	if err := updateFromIni(conf); err != nil {
		return nil, errors.Wrap(err, "error while reading ini config")
	}

	updateFromFlags(conf)

	return conf, nil
}

func createFromEnv() *Config {
	return &Config{
		Server: Server{
			Addr:          os.Getenv(envAddr),
			TlsCert:       os.Getenv(envTlsCert),
			TlsKey:        os.Getenv(envTlsKey),
			AllowedOrigin: os.Getenv(envAllowedOrigin),
		},
		Logs: Logs{
			Level:  os.Getenv(envLogsLevel),
			Format: os.Getenv(envLogsFormat),
		},
		Apple: Apple{
			Cert:   os.Getenv(envAppleCert),
			Bundle: os.Getenv(envAppleBundle),
		},
	}
}

func updateFromIni(conf *Config) error {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return nil
	}

	confIni, err := ini.Load(configFile)
	if err != nil {
		return errors.Wrapf(err, "error while trying to read a config file: %s", configFile)
	}

	addrIni := confIni.Section("server").Key("addr").String()
	if addrIni != "" {
		conf.Server.Addr = addrIni
	}

	certIni := confIni.Section("server").Key("tls_cert").String()
	if certIni != "" {
		conf.Server.TlsCert = certIni
	}

	keyIni := confIni.Section("server").Key("tls_key").String()
	if keyIni != "" {
		conf.Server.TlsKey = keyIni
	}

	allowedOriginIni := confIni.Section("server").Key("allowed_origin").String()
	if allowedOriginIni != "" {
		conf.Server.AllowedOrigin = allowedOriginIni
	}

	logsLevelIni := confIni.Section("logs").Key("level").String()
	if logsLevelIni != "" {
		conf.Logs.Level = logsLevelIni
	}

	logsFormatIni := confIni.Section("logs").Key("format").String()
	if logsFormatIni != "" {
		conf.Logs.Level = logsFormatIni
	}

	return nil
}

func updateFromFlags(conf *Config) {
	if addr != "" {
		conf.Server.Addr = addr
	}

	if tlsCert != "" {
		conf.Server.TlsCert = tlsCert
	}

	if tlsKey != "" {
		conf.Server.TlsKey = tlsKey
	}

	if allowedOrigin != "" {
		conf.Server.AllowedOrigin = allowedOrigin
	}

	if loggingLevel != "" {
		conf.Logs.Level = loggingLevel
	}

	if loggingFormat != "" {
		conf.Logs.Format = loggingFormat
	}
}
