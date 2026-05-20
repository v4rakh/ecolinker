package config

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	golog "log"
	"os"
	"time"

	"git.myservermanager.com/varakh/ecolinker/internal/server/constant"
	"git.myservermanager.com/varakh/ecolinker/internal/server/validate"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sethvargo/go-envconfig"
	zerologgorm "github.com/skynet2/zerolog-gorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

//go:embed migrations_postgres/*.sql
var migrationPostgresFS embed.FS

type Logging struct {
	Encoding              constant.ConfigLogEncoding    `env:"LOGGING_ENCODING,default=console"                   validate:"required,oneof=json console"`
	EncodingColorize      bool                          `env:"LOGGING_ENCODING_COLORIZE,default=false"`
	EncodingErrorKey      string                        `env:"LOGGING_ENCODING_ERROR_KEY,default=error"           validate:"required"`
	EncodingFileKey       string                        `env:"LOGGING_ENCODING_FILE_KEY,default=file"             validate:"required"`
	EncodingFuncKey       string                        `env:"LOGGING_ENCODING_FUNC_KEY,default=func"             validate:"required"`
	EncodingLevelKey      string                        `env:"LOGGING_ENCODING_LEVEL_KEY,default=level"           validate:"required"`
	EncodingMessageKey    string                        `env:"LOGGING_ENCODING_MESSAGE_KEY,default=msg"           validate:"required"`
	EncodingStacktraceKey string                        `env:"LOGGING_ENCODING_STACKTRACE_KEY,default=stacktrace" validate:"required"`
	EncodingTimeEncoder   constant.ConfigLogTimeEncoder `env:"LOGGING_ENCODING_TIME_ENCODER,default=rfc3339"      validate:"required,oneof=epoch epochmillis epochnanos iso8601 rfc3339 rfc3339nano"`
	EncodingTimeKey       string                        `env:"LOGGING_ENCODING_TIME_KEY,default=ts"               validate:"required"`
	Level                 string                        `env:"LOGGING_LEVEL,default=info"                         validate:"required,oneof=trace debug info warn error fatal panic disabled"`
	LevelRequests         string                        `env:"LOGGING_LEVEL_REQUESTS,default=disabled"            validate:"required,oneof=trace debug info warn error fatal panic disabled"`
}

type App struct {
	TimeZone    string `env:"TZ,default=Etc/UTC"        validate:"required"`
	Development bool   `env:"DEVELOPMENT,default=false"`
}

type Auth struct {
	AuthMethod           constant.ConfigAuthMode `env:"AUTH_MODE,default=none"                         validate:"required,oneof=none basic_single basic_credentials"`
	BasicAuthUser        string                  `env:"BASIC_AUTH_USER"                                validate:"required_if=AuthMethod basic_single"`
	BasicAuthPassword    string                  `env:"BASIC_AUTH_PASSWORD"                            validate:"required_if=AuthMethod basic_single"`
	BasicAuthCredentials map[string]string       `env:"BASIC_AUTH_CREDENTIALS,separator=|,delimiter=;" validate:"required_if=AuthMethod basic_credentials"`
}

type EcoFlow struct {
	URL                      string        `env:"ECOFLOW_URL,default=https://api-e.ecoflow.com"  validate:"required"`
	AccessKey                string        `env:"ECOFLOW_ACCESS_KEY,required"                    validate:"required"`
	SecretKey                string        `env:"ECOFLOW_SECRET_KEY,required"                    validate:"required"`
	MqttEnabled              bool          `env:"ECOFLOW_MQTT_ENABLED,default=true"`
	MqttMaxReconnectInterval time.Duration `env:"ECOFLOW_MQTT_MAX_RECONNECT_INTERVAL,default=1h" validate:"gte=0"`
	MqttWaitDisconnect       uint          `env:"ECOFLOW_MQTT_WAIT_DISCONNECT,default=1000"      validate:"numeric,gte=0"`
	MqttDebugMessages        bool          `env:"ECOFLOW_MQTT_DEBUG_MESSAGES,default=false"`
}

type MqttForward struct {
	Enabled              bool          `env:"MQTT_FORWARD_ENABLED,default=false"`
	Protocol             string        `env:"MQTT_FORWARD_PROTOCOL,default=tcp"              validate:"required_if=Enabled true,oneof=tcp ssl ws mqtts"`
	Host                 string        `env:"MQTT_FORWARD_HOST"                              validate:"required_if=Enabled true"`
	Port                 int           `env:"MQTT_FORWARD_PORT,default=1883"                 validate:"required_if=Enabled true,gte=1"`
	MaxReconnectInterval time.Duration `env:"MQTT_FORWARD_MAX_RECONNECT_INTERVAL,default=1h" validate:"gte=0"`
	WaitDisconnect       uint          `env:"MQTT_FORWARD_WAIT_DISCONNECT,default=1000"      validate:"numeric,gte=0"`
	Username             string        `env:"MQTT_FORWARD_USERNAME"`
	Password             string        `env:"MQTT_FORWARD_PASSWORD"`
}

type Server struct {
	Port              int           `env:"SERVER_PORT,default=8080"               validate:"gte=1"`
	Listen            string        `env:"SERVER_LISTEN"`
	BasePath          string        `env:"SERVER_BASE_PATH,default=/"             validate:"required"`
	TlsEnabled        bool          `env:"SERVER_TLS_ENABLED,default=false"`
	TlsCertPath       string        `env:"SERVER_TLS_CERT_PATH"`
	TlsKeyPath        string        `env:"SERVER_TLS_KEY_PATH"`
	Timeout           time.Duration `env:"SERVER_TIMEOUT,default=10s"             validate:"gte=0"`
	ReadHeaderTimeout time.Duration `env:"SERVER_READ_HEADER_TIMEOUT,default=30s" validate:"gte=0"`
}

type Cors struct {
	AllowCredentials bool     `env:"CORS_ALLOW_CREDENTIALS,default=true"`
	AllowOrigins     []string `env:"CORS_ALLOW_ORIGINS,default=*"`
	AllowMethods     []string `env:"CORS_ALLOW_METHODS,default=HEAD,GET,POST,PUT,PATCH,DELETE,OPTIONS"`
	AllowHeaders     []string `env:"CORS_ALLOW_HEADERS,default=Authorization,Content-Type"`
	ExposeHeaders    []string `env:"CORS_EXPOSE_HEADERS,default=*"`
}

type Database struct {
	Type             constant.ConfigDatabaseType `env:"DB_TYPE,default=postgres"           validate:"required,oneof=postgres"`
	MigrationEnabled bool                        `env:"DB_MIGRATION_ENABLED,default=true"`
	PostgresHost     string                      `env:"DB_POSTGRES_HOST,default=localhost" validate:"required_if=Type postgres"`
	PostgresPort     int                         `env:"DB_POSTGRES_PORT,default=5432"      validate:"required_if=Type postgres"`
	PostgresName     string                      `env:"DB_POSTGRES_NAME"                   validate:"required_if=Type postgres"`
	PostgresTimeZone string                      `env:"DB_POSTGRES_TZ,default=Etc/UTC"     validate:"required_if=Type postgres"`
	PostgresUser     string                      `env:"DB_POSTGRES_USER"                   validate:"required_if=Type postgres"`
	PostgresPassword string                      `env:"DB_POSTGRES_PASSWORD"               validate:"required_if=Type postgres"`
}

type Lock struct {
	RedisEnabled        bool          `env:"LOCK_REDIS_ENABLED,default=false"`
	RedisHost           string        `env:"LOCK_REDIS_HOST,default=localhost"        validate:"required_if=RedisEnabled true"`
	RedisPort           int           `env:"LOCK_REDIS_PORT,default=6379"             validate:"required_if=RedisEnabled true,numeric,gte=1"`
	RedisDbName         int           `env:"LOCK_REDIS_DB_NAME,default=0"             validate:"numeric,gte=0"`
	RedisUsername       string        `env:"LOCK_REDIS_USERNAME"`
	RedisPassword       string        `env:"LOCK_REDIS_PASSWORD"`
	RedisTaskTries      int           `env:"LOCK_REDIS_TASK_LOCK_TRIES,default=1"     validate:"required_if=RedisEnabled true,numeric,gte=1"`
	RedisTaskLockAtMost time.Duration `env:"LOCK_REDIS_TASK_LOCK_AT_MOST,default=30s" validate:"required_if=RedisEnabled true,gte=0"`
	RedisTaskRetryDelay time.Duration `env:"LOCK_REDIS_TASK_RETRY_DELAY,default=5s"   validate:"required_if=RedisEnabled true,gte=0"`
	RedisUrl            string
}

type Prometheus struct {
	Enabled            bool          `env:"PROMETHEUS_ENABLED,default=true"`
	Port               int           `env:"PROMETHEUS_PORT,default=8080"                  validate:"required_if=Enabled true,gte=1"`
	Listen             string        `env:"PROMETHEUS_LISTEN"`
	BasePath           string        `env:"PROMETHEUS_BASE_PATH,default=/"                validate:"required_if=Enabled true"`
	Path               string        `env:"PROMETHEUS_METRICS_PATH,default=/metrics"      validate:"required_if=Enabled true"`
	SecureTokenEnabled bool          `env:"PROMETHEUS_SECURE_TOKEN_ENABLED,default=false"`
	SecureToken        string        `env:"PROMETHEUS_SECURE_TOKEN"                       validate:"required_if=Enabled true SecureTokenEnabled true"`
	RefreshInterval    time.Duration `env:"PROMETHEUS_REFRESH_INTERVAL,default=30s"       validate:"required,gte=0"`
}

type Configuration struct {
	App         *App
	Auth        *Auth
	Cors        *Cors
	Database    *Database
	EcoFlow     *EcoFlow
	Lock        *Lock
	Logging     *Logging
	MqttForward *MqttForward
	Prometheus  *Prometheus
	Server      *Server
}

func LoadFromEnvironment(ctx context.Context) (*Configuration, *gorm.DB) {
	var err error

	// bootstrap logging (configured independently and required before any other action)
	var lc Logging
	if err = envconfig.Process(ctx, &lc); err != nil {
		golog.Fatalf("Cannot load logging configuration from environment. Reason: %v", err)
	}
	if err = validate.ValidOrError(lc); err != nil {
		golog.Fatalf("Cannot validate logging configuration. Reason: %s", err)
	}

	configureLogger(&lc)

	// load configuration and validate from environment
	var c Configuration
	if err = envconfig.Process(ctx, &c); err != nil {
		log.Fatal().Msgf("Cannot load configuration from environment. Reason: %v", err)
	}
	if err = validate.ValidOrError(c); err != nil {
		log.Fatal().Msgf("Cannot validate configuration. Reason: %s", err.Error())
	}

	var db *gorm.DB
	var migrationDriver database.Driver
	var migrationDatabaseName string
	var migrationFS source.Driver

	log.Info().Msgf("Using database type '%s'", c.Database.Type)

	if constant.ConfigDatabaseTypePostgres == c.Database.Type {
		host := c.Database.PostgresHost
		port := c.Database.PostgresPort
		dbUser := c.Database.PostgresUser
		dbPass := c.Database.PostgresPassword
		dbName := c.Database.PostgresName
		dbTZ := c.Database.PostgresTimeZone
		migrationDatabaseName = dbName

		dsn := fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v sslmode=disable TimeZone=%v", host, dbUser, dbPass, dbName, port, dbTZ)

		gormLog := zerologgorm.NewLogger(
			zerologgorm.WithDefaultLogLevel(zerolog.DebugLevel),
			zerologgorm.WithSlowThreshold(500*time.Millisecond),
			zerologgorm.WithLogParams(),
			zerologgorm.WithIgnoreNotFoundError(),
		)

		if db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormLog}); err != nil {
			log.Fatal().Msgf("Could not setup database: %v", err)
		}

		var sqlDb *sql.DB
		if sqlDb, err = db.DB(); err != nil {
			log.Fatal().Msgf("Could not retrieve database: %v", err)
		}

		if err = sqlDb.PingContext(context.Background()); err != nil {
			log.Fatal().Msgf("Could not connect to database: %v", err)
		}

		if migrationDriver, err = migratepostgres.WithInstance(sqlDb, &migratepostgres.Config{}); err != nil {
			log.Fatal().Msgf("Could not create migration driver: %v", err)
		}

		if migrationFS, err = iofs.New(migrationPostgresFS, "migrations_postgres"); err != nil {
			log.Fatal().Msgf("Could not create migration source: %v", err)
		}
	}

	if db == nil {
		log.Fatal().Msgf("Could not setup database")
	}

	if !c.Database.MigrationEnabled {
		log.Warn().Msg("Database schema migration is disabled and not executed automatically. Make sure to run them manually, otherwise the application might misbehave. You can safely ignore this warning if application is started in high availability mode and you're sure necessary database schema already exists.")
	} else {
		var migrator *migrate.Migrate
		if migrator, err = migrate.NewWithInstance("iofs", migrationFS, migrationDatabaseName, migrationDriver); err != nil {
			log.Fatal().Msgf("Could not create database migration instance: %v", err)
		}

		var migrationVersion uint
		var migrationVersionDirty bool
		if migrationVersion, migrationVersionDirty, err = migrator.Version(); err != nil {
			if errors.Is(err, migrate.ErrNilVersion) {
				log.Info().Msgf("Database migration schema is uninitialized")
			} else {
				log.Fatal().Msgf("Could not retrieve database migration version: %v", err)
			}
		} else {
			log.Info().Msgf("Previous database migration version is '%d' (dirty '%v')", migrationVersion, migrationVersionDirty)
		}

		log.Info().Msgf("Applying necessary database migration steps...")
		if err = migrator.Up(); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				log.Info().Msgf("No database schema changes detected")
			} else {
				log.Fatal().Msgf("Could not migrate database schema: %v", err)
			}
		}

		log.Info().Msgf("Applied all necessary database migration steps successfully")
	}

	// custom defaults and validation
	if c.Lock.RedisEnabled {
		if c.Lock.RedisUsername != "" && c.Lock.RedisPassword != "" {
			c.Lock.RedisUrl = fmt.Sprintf("redis://%s:%s@%s:%d/%d", c.Lock.RedisUsername, c.Lock.RedisPassword, c.Lock.RedisHost, c.Lock.RedisPort, c.Lock.RedisDbName)
		} else {
			c.Lock.RedisUrl = fmt.Sprintf("redis://%s:%d/%d", c.Lock.RedisHost, c.Lock.RedisPort, c.Lock.RedisDbName)
		}
	}

	log.Info().Msgf("Configuration: App %+v", c.App)
	log.Info().Msgf("Configuration: Auth ***REDACTED***")
	log.Info().Msgf("Configuration: Cors %+v", c.Cors)
	log.Info().Msgf("Configuration: Database ***REDACTED***")
	log.Info().Msgf("Configuration: EcoFlow ***REDACTED***")
	log.Info().Msgf("Configuration: Lock ***REDACTED***")
	log.Info().Msgf("Configuration: Logging %+v", lc)
	log.Info().Msgf("Configuration: Prometheus ***REDACTED***")
	log.Info().Msgf("Configuration: MqttForward ***REDACTED***")
	log.Info().Msgf("Configuration: Server %+v", c.Server)

	return &c, db
}

func configureLogger(cfg *Logging) {
	var level zerolog.Level
	var err error
	if level, err = zerolog.ParseLevel(cfg.Level); err != nil {
		golog.Fatalf("Cannot parse logging level: %v", err)
	}
	zerolog.SetGlobalLevel(level)

	zerolog.CallerFieldName = cfg.EncodingFuncKey
	zerolog.ErrorFieldName = cfg.EncodingErrorKey
	zerolog.ErrorStackFieldName = cfg.EncodingStacktraceKey
	zerolog.LevelFieldName = cfg.EncodingLevelKey
	zerolog.MessageFieldName = cfg.EncodingMessageKey
	zerolog.TimestampFieldName = cfg.EncodingTimeKey

	var timeEncoders = map[constant.ConfigLogTimeEncoder]string{
		constant.ConfigLogTimeEncoderEpoch:       zerolog.TimeFormatUnix,
		constant.ConfigLogTimeEncoderEpochmillis: zerolog.TimeFormatUnixMs,
		constant.ConfigLogTimeEncoderEpochnanos:  zerolog.TimeFormatUnixNano,
		constant.ConfigLogTimeEncoderIso8601:     "2006-01-02T15:04:05-0700",
		constant.ConfigLogTimeEncoderRfc3339:     time.RFC3339,
		constant.ConfigLogTimeEncoderRfc3339nano: time.RFC3339Nano,
	}
	if enc, ok := timeEncoders[cfg.EncodingTimeEncoder]; ok {
		zerolog.TimeFieldFormat = enc
	}

	if constant.ConfigLogEncodingJson == cfg.Encoding {
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()
	} else {
		log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: zerolog.TimeFieldFormat, NoColor: !cfg.EncodingColorize}).With().Timestamp().Caller().Logger()
	}
}
