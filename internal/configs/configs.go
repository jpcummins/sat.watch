package configs

import (
	"context"
	"os"
	"sync"

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
	SmtpPort     int    `env:"SMTP_PORT"`
	SmtpUser     string `env:"SMTP_USER"`
	SmtpPassword string `env:"SMTP_PASSWORD"`
	SmtpFrom     string `env:"SMTP_FROM"`
	URL          string `env:"URL"`
	Version      string `env:"APP_VERSION"`
	RPCHost      string `env:"RPCHOST"`
	RPCUser      string `env:"RPCUSER"`
	RPCPassword  string `env:"RPCPASSWORD"`

	mu sync.RWMutex
}

type dbConfig struct {
	SmtpHost     *string
	SmtpPort     *int
	SmtpUser     *string
	SmtpPassword *string
	SmtpFrom     *string
}

func readDBConfig(dbURL string) (dbConfig, error) {
	var cfg dbConfig
	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		return cfg, err
	}
	defer conn.Close(context.Background())

	err = conn.QueryRow(context.Background(),
		"SELECT smtp_host, smtp_port, smtp_user, smtp_password, smtp_from FROM config LIMIT 1").
		Scan(&cfg.SmtpHost, &cfg.SmtpPort, &cfg.SmtpUser, &cfg.SmtpPassword, &cfg.SmtpFrom)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}

func InitConifg(path string) (*Config, error) {
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
	if !cfg.IsSMTPConfigured() {
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
			if cfg.SmtpFrom == "" && dbCfg.SmtpFrom != nil {
				cfg.SmtpFrom = *dbCfg.SmtpFrom
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
	return &cfg, err
}

func (c *Config) UpdateSMTPConfig(host string, port int, user string, password string, from string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := pgx.Connect(context.Background(), c.DatabaseUrl)
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(context.Background(),
		`UPDATE config 
		SET smtp_host = $1, 
		    smtp_port = $2, 
		    smtp_user = $3, 
		    smtp_password = $4,
		    smtp_from = $5,
		    updated_at = now()`,
		host, port, user, password, from)
	if err != nil {
		return err
	}

	c.SmtpHost = host
	c.SmtpPort = port
	c.SmtpUser = user
	c.SmtpPassword = password
	c.SmtpFrom = from

	return nil
}

func (c *Config) UpdateSMTPHost(host string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := pgx.Connect(context.Background(), c.DatabaseUrl)
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(context.Background(),
		`UPDATE config 
		SET smtp_host = $1,
		    updated_at = now()`,
		host)
	if err != nil {
		return err
	}

	c.SmtpHost = host
	return nil
}

func (c *Config) UpdateSMTPPort(port int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := pgx.Connect(context.Background(), c.DatabaseUrl)
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(context.Background(),
		`UPDATE config 
		SET smtp_port = $1,
		    updated_at = now()`,
		port)
	if err != nil {
		return err
	}

	c.SmtpPort = port
	return nil
}

func (c *Config) UpdateSMTPUser(user string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := pgx.Connect(context.Background(), c.DatabaseUrl)
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(context.Background(),
		`UPDATE config 
		SET smtp_user = $1,
		    updated_at = now()`,
		user)
	if err != nil {
		return err
	}

	c.SmtpUser = user
	return nil
}

func (c *Config) UpdateSMTPPassword(password string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := pgx.Connect(context.Background(), c.DatabaseUrl)
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(context.Background(),
		`UPDATE config 
		SET smtp_password = $1,
		    updated_at = now()`,
		password)
	if err != nil {
		return err
	}

	c.SmtpPassword = password
	return nil
}

func (c *Config) GetSMTPConfig() (host string, port int, user string, password string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.SmtpHost, c.SmtpPort, c.SmtpUser, c.SmtpPassword
}

func (c *Config) IsSMTPConfigured() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.SmtpHost != "" && c.SmtpPort != 0 && c.SmtpUser != "" && c.SmtpPassword != "" && c.SmtpFrom != ""
}
