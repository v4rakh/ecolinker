package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"git.myservermanager.com/varakh/ecolinker/internal/meta"
	"git.myservermanager.com/varakh/ecolinker/internal/server/config"
	"git.myservermanager.com/varakh/ecolinker/internal/server/constant"
	"git.myservermanager.com/varakh/ecolinker/internal/server/handler"
	"git.myservermanager.com/varakh/ecolinker/internal/server/repository"
	"git.myservermanager.com/varakh/ecolinker/internal/server/service"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"go.uber.org/automaxprocs/maxprocs"
)

type Server struct {
}

func New() *Server {
	return &Server{}
}

func (s *Server) Start(ctx context.Context) {
	var err error

	// configuration init
	cfg, db := config.LoadFromEnvironment(ctx)

	log.Info().Msgf("Starting %s %s", meta.Name, meta.Version)

	// adhere to GOMAXPROCS, but silence default output
	_, _ = maxprocs.Set(maxprocs.Logger(nil))
	log.Debug().Msgf("GOMAXPROCS '%d'", runtime.GOMAXPROCS(0))

	// set gin mode derived
	if cfg.App.Development {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	corsMiddleware := middlewareCors(cfg.Cors)
	loggingMiddleware := middlewareLogging(cfg.Logging)
	recoveryMiddleware := middlewarePanicRecoveryHandler(cfg.Logging)
	errorMiddleware := middlewareErrorTransformer()

	// routers init
	appRouter := s.newEngine(loggingMiddleware, recoveryMiddleware, corsMiddleware, middlewareAppName(), middlewareAppVersion(), errorMiddleware)
	promRouter := s.newEngine(loggingMiddleware, recoveryMiddleware, errorMiddleware)

	// repositories init
	deviceRepo := repository.NewDeviceDbRepo(db)
	mqttSubRepo := repository.NewMqttSubscriptionDbRepo(db)
	collectorRepo := repository.NewCollectorDbRepo(db)

	// services init
	lockService := service.NewLockMemService()
	if cfg.Lock.RedisEnabled {
		var e error
		if lockService, e = service.NewLockRedisService(cfg.Lock); e != nil {
			log.Fatal().Err(e).Msg("Failed to create lock service")
		}
	}

	var taskService *service.TaskService
	if taskService, err = service.NewTaskService(lockService, cfg.App, cfg.Lock); err != nil {
		log.Fatal().Err(err).Msg("Task service creation failed")
	}

	ecoFlowHttpService := service.NewEcoFlowHttpService(cfg.EcoFlow.AccessKey, cfg.EcoFlow.SecretKey, cfg.EcoFlow.URL)

	mqttForwardService := service.NewMqttForwardService(taskService, cfg.MqttForward)
	if err = mqttForwardService.Init(); err != nil {
		log.Fatal().Err(err).Msg("MQTT forward service initialization failed")
	}

	separatePromServer := cfg.Prometheus.Enabled && cfg.Prometheus.Port != cfg.Server.Port
	var prometheusService *service.PrometheusService
	if cfg.Prometheus.Enabled && separatePromServer {
		prometheusService = service.NewPrometheusService(promRouter, cfg.Prometheus)
		log.Info().Msg("Starting separate Prometheus server")
	} else if cfg.Prometheus.Enabled && !separatePromServer {
		prometheusService = service.NewPrometheusService(appRouter, cfg.Prometheus)
		log.Info().Msg("Starting embedded Prometheus server")
	}
	if cfg.Prometheus.Enabled {
		// always instrument tracking for the meta router
		appRouter.Use(prometheusService.GetProm().Instrument())
	}

	mqttSubReadService := service.NewMqttSubscriptionReadService(mqttSubRepo)

	ecoFlowMqttService := service.NewEcoFlowMqttService(ecoFlowHttpService, mqttSubReadService, mqttForwardService, prometheusService, taskService, cfg.EcoFlow)
	if err = ecoFlowMqttService.Init(); err != nil {
		log.Fatal().Err(err).Msg("EcoFlow service initialization failed")
	}

	ecoFlowMqttTask := service.NewEcoFlowMqttTask(ecoFlowMqttService, mqttForwardService, prometheusService, taskService, cfg.EcoFlow)
	mqttSubWriteService := service.NewMqttSubscriptionWriteService(mqttSubReadService, ecoFlowMqttTask, mqttSubRepo)
	deviceService := service.NewDeviceService(mqttSubReadService, ecoFlowMqttTask, deviceRepo)

	collectorService := service.NewCollectorService(ecoFlowHttpService, mqttForwardService, taskService, prometheusService, collectorRepo)
	if err = collectorService.Init(); err != nil {
		log.Fatal().Err(err).Msg("Collector service initialization failed")
	}

	prometheusTask := service.NewPrometheusTask(cfg.Prometheus, ecoFlowMqttService, mqttForwardService, prometheusService, taskService)
	if err = prometheusTask.Init(); err != nil {
		log.Fatal().Err(err).Msg("Task prometheus task initialization failed")
	}
	taskService.Start()

	// handlers init
	infoHandler := handler.NewInfoHandler(cfg.App)
	healthHandler := handler.NewHealthHandler()
	deviceHandler := handler.NewDeviceHandler(deviceService)
	mqttSubHandler := handler.NewMqttSubscriptionHandler(mqttSubReadService, mqttSubWriteService)
	collectorHandler := handler.NewCollectorHandler(collectorService)
	ecoFlowHandler := handler.NewEcoFlowHandler(ecoFlowHttpService, ecoFlowMqttService)

	apiPublicGroup := appRouter.Group(cfg.Server.BasePath + "/api/v1")
	apiPublicGroup.GET("/health", healthHandler.Status)
	apiPublicGroup.GET("/info", infoHandler.Status)

	var authMethodHandler gin.HandlerFunc

	if constant.ConfigAuthModeBasicSingle == cfg.Auth.AuthMethod {
		authMethodHandler = gin.BasicAuth(gin.Accounts{
			cfg.Auth.BasicAuthUser: cfg.Auth.BasicAuthPassword,
		})
	} else if constant.ConfigAuthModeBasicCredentials == cfg.Auth.AuthMethod {
		authMethodHandler = gin.BasicAuth(cfg.Auth.BasicAuthCredentials)
	} else if constant.ConfigAuthModeNone == cfg.Auth.AuthMethod {
		authMethodHandler = func(c *gin.Context) {}
	} else {
		log.Fatal().Msg("No valid auth mode found")
	}

	apiAuthGroup := appRouter.Group(cfg.Server.BasePath+"api/v1", authMethodHandler)

	apiAuthGroup.GET("/devices", deviceHandler.GetAll)
	apiAuthGroup.GET("/devices/:sn", deviceHandler.Get)
	apiAuthGroup.POST("/devices", middlewareEnforceJsonContentType(), deviceHandler.Create)
	apiAuthGroup.PUT("/devices", middlewareEnforceJsonContentType(), deviceHandler.Update)
	apiAuthGroup.DELETE("/devices/:sn", deviceHandler.Delete)

	apiAuthGroup.GET("/mqtt-subscriptions", mqttSubHandler.Get)
	apiAuthGroup.POST("/mqtt-subscriptions", middlewareEnforceJsonContentType(), mqttSubHandler.Create)
	apiAuthGroup.PUT("/mqtt-subscriptions/:id", middlewareEnforceJsonContentType(), mqttSubHandler.Update)
	apiAuthGroup.DELETE("/mqtt-subscriptions/:id", mqttSubHandler.Delete)

	apiAuthGroup.GET("/collectors", collectorHandler.Get)
	apiAuthGroup.POST("/collectors", middlewareEnforceJsonContentType(), collectorHandler.Create)
	apiAuthGroup.PUT("/collectors/:id", middlewareEnforceJsonContentType(), collectorHandler.Update)
	apiAuthGroup.DELETE("/collectors/:id", collectorHandler.Delete)
	apiAuthGroup.POST("/collectors/:id/invoke", collectorHandler.Invoke)

	apiAuthGroup.GET("/ecoflow/status", ecoFlowHandler.BrokerStatus)
	apiAuthGroup.GET("/ecoflow/devices", ecoFlowHandler.Devices)
	apiAuthGroup.POST("/ecoflow/devices/:sn", ecoFlowHandler.Parameters)
	apiAuthGroup.GET("/ecoflow/devices/:sn", ecoFlowHandler.ParametersAll)
	apiAuthGroup.GET("/ecoflow/devices/:sn/history", ecoFlowHandler.History)
	apiAuthGroup.GET("/ecoflow/devices/:sn/batteries", ecoFlowHandler.Batteries)

	// start servers (run in separate goroutines)
	appSrv := s.newServer(appRouter, fmt.Sprintf("%s:%d", cfg.Server.Listen, cfg.Server.Port), cfg.Server.ReadHeaderTimeout)
	prometheusSrv := s.newServer(promRouter, fmt.Sprintf("%s:%d", cfg.Prometheus.Listen, cfg.Prometheus.Port), cfg.Server.ReadHeaderTimeout)

	s.startServer(appSrv, cfg.Server)

	if separatePromServer {
		s.startServer(prometheusSrv, cfg.Server)
	}

	// gracefully handle shut down
	// Wait for interrupt signal to gracefully shut down the server with
	// a timeout of x seconds.
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall. SIGKILL but cannot be caught, thus no need to add
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down...")

	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, cfg.Server.Timeout)
	defer timeoutCancel()

	shutdownDone := make(chan struct{})
	go func() {
		taskService.Stop()
		ecoFlowMqttService.Disconnect()
		mqttForwardService.Disconnect()
		s.stopServer(ctx, appSrv)
		s.stopServer(ctx, prometheusSrv)
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
		log.Info().Msg("Exited")
	case <-timeoutCtx.Done():
		log.Info().Msgf("Shutdown timeout of '%v' expired, exiting forcefully...", cfg.Server.Timeout)
		os.Exit(1)
	}
}

func (s *Server) newServer(r *gin.Engine, address string, readHeaderTimeout time.Duration) *http.Server {
	if r == nil || address == "" {
		log.Fatal().Msg("Failed to create server, engine or address is nil")
		return nil
	}

	return &http.Server{
		Addr:              address,
		Handler:           r,
		ReadHeaderTimeout: readHeaderTimeout,
	}
}

func (s *Server) startServer(h *http.Server, cfg *config.Server) {
	go func() {
		var e error
		log.Info().Msgf("Server listening on '%s'", h.Addr)

		if cfg.TlsEnabled {
			e = h.ListenAndServeTLS(cfg.TlsCertPath, cfg.TlsKeyPath)
		} else {
			e = h.ListenAndServe()
		}

		if e != nil && !errors.Is(e, http.ErrServerClosed) {
			log.Fatal().Err(e).Msg("Server cannot be started")
		}
	}()
}

func (s *Server) stopServer(ctx context.Context, h *http.Server) {
	if h == nil {
		return
	}

	if err := h.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Shutdown failed, exited directly")
	}

	log.Info().Msgf("Shutdown for '%s' complete", h.Addr)
}

func (s *Server) newEngine(middleware ...gin.HandlerFunc) *gin.Engine {
	r := gin.New()

	for _, m := range middleware {
		r.Use(m)
	}

	r.NoMethod(middlewareGlobalMethodNotAllowed())
	r.NoRoute(middlewareGlobalNotFound())

	return r
}
