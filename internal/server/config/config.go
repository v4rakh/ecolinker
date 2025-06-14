package config

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"git.myservermanager.com/varakh/ecolinker/internal/server/constant"
	"git.myservermanager.com/varakh/ecolinker/internal/server/validate"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/sethvargo/go-envconfig"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"moul.io/zapgorm2"
	"time"
)

//go:embed migrations_postgres/*.sql
var migrationPostgresFS embed.FS

type Logging struct {
	Debug                   bool                              `env:"DEBUG,default=false"`
	Development             bool                              `env:"DEVELOPMENT,default=false"`
	Encoding                constant.ConfigLogEncoding        `env:"LOGGING_ENCODING,default=console" validate:"required,oneof=json console"`
	EncodingCallerEncoder   constant.ConfigLogCallerEncoder   `env:"LOGGING_ENCODING_CALLER_ENCODER,default=short" validate:"required,oneof=full short"`
	EncodingDurationEncoder constant.ConfigLogDurationEncoder `env:"LOGGING_ENCODING_DURATION_ENCODER,default=seconds" validate:"required,oneof=seconds nanos millis string"`
	EncodingLevelEncoder    constant.ConfigLogLevelEncoder    `env:"LOGGING_ENCODING_LEVEL_ENCODER,default=capital" validate:"required,oneof=lowercase lowercasecolor capital capitalcolor"`
	EncodingLevelKey        string                            `env:"LOGGING_ENCODING_LEVEL_KEY,default=level" validate:"required_if=Encoding json"`
	EncodingMessageKey      string                            `env:"LOGGING_ENCODING_MESSAGE_KEY,default=msg" validate:"required_if=Encoding json"`
	EncodingStacktraceKey   string                            `env:"LOGGING_ENCODING_STACKTRACE_KEY,default=stacktrace" validate:"required_if=Encoding json"`
	EncodingTimeEncoder     constant.ConfigLogTimeEncoder     `env:"LOGGING_ENCODING_TIME_ENCODER,default=rfc3339" validate:"required,oneof=epoch epochmillis epochnanos iso8601 rfc3339 rfc3339nano"`
	EncodingTimeKey         string                            `env:"LOGGING_ENCODING_TIME_KEY,default=ts" validate:"required_if=Encoding json"`
	Level                   string                            `env:"LOGGING_LEVEL,default=info" validate:"required,oneof=debug info warn error dpanic panic fatal"`
	UTC                     bool                              `env:"LOGGING_UTC"`
}

type App struct {
	TimeZone    string `env:"TZ,default=Etc/UTC" validate:"required"`
	Development bool   `env:"DEVELOPMENT,default=false"`
}

type Auth struct {
	AuthMethod           constant.ConfigAuthMode `env:"AUTH_MODE,default=none" validate:"required,oneof=none basic_single basic_credentials"`
	BasicAuthUser        string                  `env:"BASIC_AUTH_USER" validate:"required_if=AuthMethod basic_single"`
	BasicAuthPassword    string                  `env:"BASIC_AUTH_PASSWORD" validate:"required_if=AuthMethod basic_single"`
	BasicAuthCredentials map[string]string       `env:"BASIC_AUTH_CREDENTIALS,separator=|,delimiter=;" validate:"required_if=AuthMethod basic_credentials"`
}

type EcoFlow struct {
	URL                      string        `env:"ECOFLOW_URL,default=https://api-e.ecoflow.com" validate:"required"`
	AccessKey                string        `env:"ECOFLOW_ACCESS_KEY,required" validate:"required"`
	SecretKey                string        `env:"ECOFLOW_SECRET_KEY,required" validate:"required"`
	MqttEnabled              bool          `env:"ECOFLOW_MQTT_ENABLED,default=true"`
	MqttMaxReconnectInterval time.Duration `env:"ECOFLOW_MQTT_MAX_RECONNECT_INTERVAL,default=1h" validate:"gte=0"`
	MqttWaitDisconnect       uint          `env:"ECOFLOW_MQTT_WAIT_DISCONNECT,default=1000" validate:"numeric,gte=0"`
	MqttDebugMessages        bool          `env:"ECOFLOW_MQTT_DEBUG_MESSAGES,default=false"`
}

type MqttForward struct {
	Enabled              bool          `env:"MQTT_FORWARD_ENABLED,default=false"`
	Protocol             string        `env:"MQTT_FORWARD_PROTOCOL,default=tcp" validate:"required_if=Enabled true,oneof=tcp ssl ws mqtts"`
	Host                 string        `env:"MQTT_FORWARD_HOST" validate:"required_if=Enabled true"`
	Port                 int           `env:"MQTT_FORWARD_PORT,default=1883" validate:"required_if=Enabled true,gte=1"`
	MaxReconnectInterval time.Duration `env:"MQTT_FORWARD_MAX_RECONNECT_INTERVAL,default=1h" validate:"gte=0"`
	WaitDisconnect       uint          `env:"MQTT_FORWARD_WAIT_DISCONNECT,default=1000" validate:"numeric,gte=0"`
	Username             string        `env:"MQTT_FORWARD_USERNAME"`
	Password             string        `env:"MQTT_FORWARD_PASSWORD"`
}

type Server struct {
	Port        int           `env:"SERVER_PORT,default=8080" validate:"gte=1"`
	Listen      string        `env:"SERVER_LISTEN"`
	BasePath    string        `env:"SERVER_BASE_PATH,default=/" validate:"required"`
	TlsEnabled  bool          `env:"SERVER_TLS_ENABLED,default=false"`
	TlsCertPath string        `env:"SERVER_TLS_CERT_PATH"`
	TlsKeyPath  string        `env:"SERVER_TLS_KEY_PATH"`
	Timeout     time.Duration `env:"SERVER_TIMEOUT,default=10s" validate:"gte=0"`
}

type Cors struct {
	AllowCredentials bool     `env:"CORS_ALLOW_CREDENTIALS,default=true"`
	AllowOrigins     []string `env:"CORS_ALLOW_ORIGINS,default=*"`
	AllowMethods     []string `env:"CORS_ALLOW_METHODS,default=HEAD,GET,POST,PUT,PATCH,DELETE,OPTIONS"`
	AllowHeaders     []string `env:"CORS_ALLOW_HEADERS,default=Authorization,Content-Type"`
	ExposeHeaders    []string `env:"CORS_EXPOSE_HEADERS,default=*"`
}

type Database struct {
	Type             constant.ConfigDatabaseType `env:"DB_TYPE,default=postgres" validate:"required,oneof=postgres"`
	MigrationEnabled bool                        `env:"DB_MIGRATION_ENABLED,default=true"`
	PostgresHost     string                      `env:"DB_POSTGRES_HOST,default=localhost" validate:"required_if=Type postgres"`
	PostgresPort     int                         `env:"DB_POSTGRES_PORT,default=5432" validate:"required_if=Type postgres"`
	PostgresName     string                      `env:"DB_POSTGRES_NAME" validate:"required_if=Type postgres"`
	PostgresTimeZone string                      `env:"DB_POSTGRES_TZ,default=Etc/UTC" validate:"required_if=Type postgres"`
	PostgresUser     string                      `env:"DB_POSTGRES_USER" validate:"required_if=Type postgres"`
	PostgresPassword string                      `env:"DB_POSTGRES_PASSWORD" validate:"required_if=Type postgres"`
}

type Lock struct {
	RedisEnabled        bool          `env:"LOCK_REDIS_ENABLED,default=false"`
	RedisHost           string        `env:"LOCK_REDIS_HOST,default=localhost" validate:"required_if=RedisEnabled true"`
	RedisPort           int           `env:"LOCK_REDIS_PORT,default=6379" validate:"required_if=RedisEnabled true,numeric,gte=1"`
	RedisDbName         int           `env:"LOCK_REDIS_DB_NAME,default=0" validate:"numeric,gte=0"`
	RedisUsername       string        `env:"LOCK_REDIS_USERNAME"`
	RedisPassword       string        `env:"LOCK_REDIS_PASSWORD"`
	RedisTaskTries      int           `env:"LOCK_REDIS_TASK_LOCK_TRIES,default=1" validate:"required_if=RedisEnabled true,numeric,gte=1"`
	RedisTaskLockAtMost time.Duration `env:"LOCK_REDIS_TASK_LOCK_AT_MOST,default=30s" validate:"required_if=RedisEnabled true,gte=0"`
	RedisTaskRetryDelay time.Duration `env:"LOCK_REDIS_TASK_RETRY_DELAY,default=5s" validate:"required_if=RedisEnabled true,gte=0"`
	RedisUrl            string
}

type Prometheus struct {
	Enabled            bool          `env:"PROMETHEUS_ENABLED,default=true"`
	Port               int           `env:"PROMETHEUS_PORT,default=8080" validate:"required_if=Enabled true,gte=1"`
	Listen             string        `env:"PROMETHEUS_LISTEN"`
	BasePath           string        `env:"PROMETHEUS_BASE_PATH,default=/" validate:"required_if=Enabled true"`
	Path               string        `env:"PROMETHEUS_METRICS_PATH,default=/metrics" validate:"required_if=Enabled true"`
	SecureTokenEnabled bool          `env:"PROMETHEUS_SECURE_TOKEN_ENABLED,default=false"`
	SecureToken        string        `env:"PROMETHEUS_SECURE_TOKEN" validate:"required_if=Enabled true SecureTokenEnabled true"`
	RefreshInterval    time.Duration `env:"PROMETHEUS_REFRESH_INTERVAL,default=30s" validate:"required,gte=0"`
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
		log.Fatalf("Cannot load logging configuration from environment. Reason: %v", err)
	}
	if err = validate.ValidOrError(lc); err != nil {
		log.Fatalf("Cannot validate logging configuration. Reason: %s", err)
	}

	var level zap.AtomicLevel
	if level, err = zap.ParseAtomicLevel(lc.Level); err != nil {
		log.Fatalf("Cannot parse logging level: %v", err)
	}

	var loggingEncoderConfig zapcore.EncoderConfig
	if constant.ConfigLogEncodingJson == lc.Encoding {
		loggingEncoderConfig = zap.NewProductionEncoderConfig()
		loggingEncoderConfig.MessageKey = lc.EncodingMessageKey
		loggingEncoderConfig.LevelKey = lc.EncodingLevelKey
		loggingEncoderConfig.TimeKey = lc.EncodingTimeKey
		loggingEncoderConfig.StacktraceKey = lc.EncodingStacktraceKey
	} else {
		loggingEncoderConfig = zap.NewDevelopmentEncoderConfig()
	}

	var levelEncoders = map[constant.ConfigLogLevelEncoder]zapcore.LevelEncoder{
		constant.ConfigLogLevelEncoderLowercase:      zapcore.LowercaseLevelEncoder,
		constant.ConfigLogLevelEncoderLowercasecolor: zapcore.LowercaseColorLevelEncoder,
		constant.ConfigLogLevelEncoderCapital:        zapcore.CapitalLevelEncoder,
		constant.ConfigLogLevelEncoderCapitalcolor:   zapcore.CapitalColorLevelEncoder,
	}
	if enc, ok := levelEncoders[lc.EncodingLevelEncoder]; ok {
		loggingEncoderConfig.EncodeLevel = enc
	}

	var timeEncoders = map[constant.ConfigLogTimeEncoder]zapcore.TimeEncoder{
		constant.ConfigLogTimeEncoderEpoch:       zapcore.EpochTimeEncoder,
		constant.ConfigLogTimeEncoderEpochmillis: zapcore.EpochMillisTimeEncoder,
		constant.ConfigLogTimeEncoderEpochnanos:  zapcore.EpochNanosTimeEncoder,
		constant.ConfigLogTimeEncoderIso8601:     zapcore.ISO8601TimeEncoder,
		constant.ConfigLogTimeEncoderRfc3339:     zapcore.RFC3339TimeEncoder,
		constant.ConfigLogTimeEncoderRfc3339nano: zapcore.RFC3339NanoTimeEncoder,
	}
	if enc, ok := timeEncoders[lc.EncodingTimeEncoder]; ok {
		loggingEncoderConfig.EncodeTime = enc
	}

	var durationEncoders = map[constant.ConfigLogDurationEncoder]zapcore.DurationEncoder{
		constant.ConfigLogDurationEncoderSeconds: zapcore.SecondsDurationEncoder,
		constant.ConfigLogDurationEncoderNanos:   zapcore.NanosDurationEncoder,
		constant.ConfigLogDurationEncoderMillis:  zapcore.MillisDurationEncoder,
		constant.ConfigLogDurationEncoderString:  zapcore.StringDurationEncoder,
	}
	if enc, ok := durationEncoders[lc.EncodingDurationEncoder]; ok {
		loggingEncoderConfig.EncodeDuration = enc
	}

	var callerEncoders = map[constant.ConfigLogCallerEncoder]zapcore.CallerEncoder{
		constant.ConfigLogCallerEncoderFull:  zapcore.FullCallerEncoder,
		constant.ConfigLogCallerEncoderShort: zapcore.ShortCallerEncoder,
	}
	if enc, ok := callerEncoders[lc.EncodingCallerEncoder]; ok {
		loggingEncoderConfig.EncodeCaller = enc
	} else {
		loggingEncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	}

	logPaths := []string{"stderr"}

	var zapConfig *zap.Config
	if lc.Debug {
		zapConfig = &zap.Config{
			Level:            level,
			Development:      lc.Development,
			Encoding:         lc.Encoding.String(),
			EncoderConfig:    loggingEncoderConfig,
			OutputPaths:      logPaths,
			ErrorOutputPaths: logPaths,
		}
	} else {
		zapConfig = &zap.Config{
			Level:       level,
			Development: lc.Development,
			Sampling: &zap.SamplingConfig{
				Initial:    100,
				Thereafter: 100,
			},
			Encoding:         lc.Encoding.String(),
			EncoderConfig:    loggingEncoderConfig,
			OutputPaths:      logPaths,
			ErrorOutputPaths: logPaths,
		}
	}

	zapLogger := zap.Must(zapConfig.Build())
	defer func(zapLogger *zap.Logger) {
		_ = zapLogger.Sync()
	}(zapLogger)
	zap.ReplaceGlobals(zapLogger)

	// load configuration and validate from environment
	var c Configuration
	if err = envconfig.Process(ctx, &c); err != nil {
		zap.L().Sugar().Fatalf("Cannot load configuration from environment. Reason: %v", err)
	}
	if err = validate.ValidOrError(c); err != nil {
		zap.L().Sugar().Fatalf("Cannot validate configuration. Reason: %s", err.Error())
	}

	gormConfig := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	if lc.Debug && c.App.Development {
		gormZapLogger := zap.Must(zapConfig.Build())
		defer func(gormZapLogger *zap.Logger) {
			_ = gormZapLogger.Sync()
		}(gormZapLogger)
		gormLogger := zapgorm2.New(gormZapLogger)
		gormConfig = &gorm.Config{Logger: gormLogger}
	}

	var db *gorm.DB
	var migrationDriver database.Driver
	var migrationDatabaseName string
	var migrationFS source.Driver

	zap.L().Sugar().Infof("Using database type '%s'", c.Database.Type)

	if constant.ConfigDatabaseTypePostgres == c.Database.Type {
		host := c.Database.PostgresHost
		port := c.Database.PostgresPort
		dbUser := c.Database.PostgresUser
		dbPass := c.Database.PostgresPassword
		dbName := c.Database.PostgresName
		dbTZ := c.Database.PostgresTimeZone
		migrationDatabaseName = dbName

		dsn := fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v sslmode=disable TimeZone=%v", host, dbUser, dbPass, dbName, port, dbTZ)
		if db, err = gorm.Open(postgres.Open(dsn), gormConfig); err != nil {
			zap.L().Sugar().Fatalf("Could not setup database: %v", err)
		}

		var sqlDb *sql.DB
		if sqlDb, err = db.DB(); err != nil {
			zap.L().Sugar().Fatalf("Could not retrieve database: %v", err)
		}

		if err = sqlDb.Ping(); err != nil {
			zap.L().Sugar().Fatalf("Could not connect to database: %v", err)
		}

		if migrationDriver, err = migratepostgres.WithInstance(sqlDb, &migratepostgres.Config{}); err != nil {
			zap.L().Sugar().Fatalf("Could not create migration driver: %v", err)
		}

		if migrationFS, err = iofs.New(migrationPostgresFS, "migrations_postgres"); err != nil {
			zap.L().Sugar().Fatalf("Could not create migration source: %v", err)
		}
	}

	if db == nil {
		zap.L().Fatal("Could not setup database")
	}

	if !c.Database.MigrationEnabled {
		zap.L().Warn("Database schema migration is disabled and not executed automatically. Make sure to run them manually, otherwise the application might misbehave. You can safely ignore this warning if application is started in high availability mode and you're sure necessary database schema already exists.")
	} else {
		var migrator *migrate.Migrate
		if migrator, err = migrate.NewWithInstance("iofs", migrationFS, migrationDatabaseName, migrationDriver); err != nil {
			zap.L().Sugar().Fatalf("Could not create database migration instance: %v", err)
		}

		var migrationVersion uint
		var migrationVersionDirty bool
		if migrationVersion, migrationVersionDirty, err = migrator.Version(); err != nil {
			if errors.Is(err, migrate.ErrNilVersion) {
				zap.L().Info("Database migration schema is uninitialized")
			} else {
				zap.L().Sugar().Fatalf("Could not retrieve database migration version: %v", err)
			}
		} else {
			zap.L().Sugar().Infof("Previous database migration version is '%d' (dirty '%v')", migrationVersion, migrationVersionDirty)
		}

		zap.L().Info("Applying necessary database migration steps...")
		if err = migrator.Up(); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				zap.L().Info("No database schema changes detected")
			} else {
				zap.L().Sugar().Fatalf("Could not migrate database schema: %v", err)
			}
		}

		zap.L().Info("Applied all necessary database migration steps successfully")
	}

	// custom defaults and validation
	if c.Lock.RedisEnabled {
		if c.Lock.RedisUsername != "" && c.Lock.RedisPassword != "" {
			c.Lock.RedisUrl = fmt.Sprintf("redis://%s:%s@%s:%d/%d", c.Lock.RedisUsername, c.Lock.RedisPassword, c.Lock.RedisHost, c.Lock.RedisPort, c.Lock.RedisDbName)
		} else {
			c.Lock.RedisUrl = fmt.Sprintf("redis://%s:%d/%d", c.Lock.RedisHost, c.Lock.RedisPort, c.Lock.RedisDbName)
		}
	}

	zap.L().Sugar().Infof("Configuration: App %+v", c.App)
	zap.L().Info("Configuration: Auth ***REDACTED***")
	zap.L().Sugar().Infof("Configuration: Cors %+v", c.Cors)
	zap.L().Info("Configuration: Database ***REDACTED***")
	zap.L().Info("Configuration: EcoFlow ***REDACTED***")
	zap.L().Info("Configuration: Lock ***REDACTED***")
	zap.L().Sugar().Infof("Configuration: Logging %+v", lc)
	zap.L().Info("Configuration: Prometheus ***REDACTED***")
	zap.L().Info("Configuration: MqttForward ***REDACTED***")
	zap.L().Sugar().Infof("Configuration: Server %+v", c.Server)

	return &c, db
}
