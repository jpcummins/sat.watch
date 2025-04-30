package configs

import (
	"context"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Environment  string `env:"ENVIRONMENT" env-default:"development"`
	DatabaseUrl  string `env:"DATABASE_URL"`
	Gap          int    `env:"GAP" env-default:"5"`
	Port         string `env:"PORT" env-default:"8080"`
	Secret       string `env:"SECRET"`
	ZMQHost      string `env:"ZMQ_HOST" env-default:"127.0.0.1"`
	ZMQPort      int    `env:"ZMQ_PORT" env-default:"5555"`
	ZMQMockHost  string `env:"ZMQMOCK_HOST" env-default:"127.0.0.1"`
	ZMQMockPort  int    `env:"ZMQMOCK_PORT" env-default:"5555"`
	ElectrumHost string `env:"ELECTRUM_HOST"`
	ElectrumPort int    `env:"ELECTRUM_PORT"`
	ElectrumSSL  bool   `env:"ELECTRUM_SSL" env-default:"false"`
	SmtpHost     string `env:"SMTP_HOST"`
	SmtpPort     int    `env:"SMTP_PORT" env-default:"25"`
	SmtpUser     string `env:"SMTP_USER"`
	SmtpPassword string `env:"SMTP_PASSWORD"`
	URL          string `env:"URL"`
	Version      string `env:"APP_VERSION"`
	RPCHost      string `env:"RPCHOST"`
	RPCUser      string `env:"RPCUSER"`
	RPCPassword  string `env:"RPCPASSWORD"`
}

type dbConfig struct {
	SmtpHost     *string
	SmtpPort     *int
	SmtpUser     *string
	SmtpPassword *string
}

func readDBConfig(dbURL string) (dbConfig, error) {
	var cfg dbConfig
	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		return cfg, err
	}
	defer conn.Close(context.Background())

	err = conn.QueryRow(context.Background(),
		"SELECT smtp_host, smtp_port, smtp_user, smtp_password FROM config LIMIT 1").
		Scan(&cfg.SmtpHost, &cfg.SmtpPort, &cfg.SmtpUser, &cfg.SmtpPassword)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}

func InitConifg(path string) (Config, error) {
	logger := log.With().Str("module", "config").Logger()
	logger.Info().Msg("initializing")
	var cfg Config
	err := cleanenv.ReadEnv(&cfg)

	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	if cfg.Environment == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
		logger = log.With().Str("module", "config").Logger()
		logger.Debug().Msg("Debug output enabled")
	} else {
		log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
		logger = log.With().Str("module", "config").Logger()
		logger.Debug().Msg("Production output enabled")
	}

	// If SMTP settings are not provided via environment variables, try to read from database
	if cfg.SmtpHost == "" || cfg.SmtpPort == 0 || cfg.SmtpUser == "" || cfg.SmtpPassword == "" {
		dbCfg, err := readDBConfig(cfg.DatabaseUrl)
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to read SMTP config from database")
		} else {
			if cfg.SmtpHost == "" && dbCfg.SmtpHost != nil {
				cfg.SmtpHost = *dbCfg.SmtpHost
			}
			if cfg.SmtpPort == 0 && dbCfg.SmtpPort != nil {
				cfg.SmtpPort = *dbCfg.SmtpPort
			}
			if cfg.SmtpUser == "" && dbCfg.SmtpUser != nil {
				cfg.SmtpUser = *dbCfg.SmtpUser
			}
			if cfg.SmtpPassword == "" && dbCfg.SmtpPassword != nil {
				cfg.SmtpPassword = *dbCfg.SmtpPassword
			}
		}
	}

	cfgCopy := cfg
	cfgCopy.DatabaseUrl = "***"
	cfgCopy.Secret = "***"
	cfgCopy.SmtpPassword = "***"
	cfgCopy.RPCPassword = "***"

	logger.Info().Interface("config", &cfgCopy).Msg("parsed config")
	logger.Info().Msg("finished initializing")
	return cfg, err
}
