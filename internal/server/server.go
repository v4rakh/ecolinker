package server

import (
	"context"
	"errors"
	"fmt"
	"git.myservermanager.com/varakh/ecolinker/internal/meta"
	"git.myservermanager.com/varakh/ecolinker/internal/server/config"
	"git.myservermanager.com/varakh/ecolinker/internal/server/constant"
	"git.myservermanager.com/varakh/ecolinker/internal/server/handler"
	"git.myservermanager.com/varakh/ecolinker/internal/server/repository"
	"git.myservermanager.com/varakh/ecolinker/internal/server/service"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"go.uber.org/automaxprocs/maxprocs"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

type Server struct {
	ctx context.Context
}

func New(ctx *context.Context) *Server {
	s := &Server{}
	if ctx == nil {
		s.ctx = context.Background()
	} else {
		s.ctx = *ctx
	}

	return s
}

func (s *Server) Start() {
	var err error

	// configuration init
	cfg, db := config.LoadFromEnvironment(s.ctx)

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
			log.Fatal().Msgf("Failed to create lock service: %+v", e)
		}
	}

	var taskService *service.TaskService
	if taskService, err = service.NewTaskService(lockService, cfg.App, cfg.Lock); err != nil {
		log.Fatal().Msgf("Task service creation failed: %v", err)
	}

	ecoFlowHttpService := service.NewEcoFlowHttpService(cfg.EcoFlow)

	mqttForwardService := service.NewMqttForwardService(taskService, cfg.MqttForward)
	if err = mqttForwardService.Init(); err != nil {
		log.Fatal().Msgf("MQTT forward service initialization failed: %v", err)
	}

	separatePromServer := cfg.Prometheus.Enabled && cfg.Prometheus.Port != cfg.Server.Port
	var prometheusService *service.PrometheusService
	if cfg.Prometheus.Enabled && separatePromServer {
		prometheusService = service.NewPrometheusService(promRouter, cfg.Prometheus)
		log.Info().Msgf("Starting separate Prometheus server")
	} else if cfg.Prometheus.Enabled && !separatePromServer {
		prometheusService = service.NewPrometheusService(appRouter, cfg.Prometheus)
		log.Info().Msgf("Starting embedded Prometheus server")
	}
	if cfg.Prometheus.Enabled {
		if err = prometheusService.Init(); err != nil {
			log.Fatal().Msgf("Prometheus service initialization failed: %v", err)
		}
		// always instrument tracking for the meta router
		appRouter.Use(prometheusService.GetProm().Instrument())
	}

	mqttSubReadService := service.NewMqttSubscriptionReadService(mqttSubRepo)

	ecoFlowMqttService := service.NewEcoFlowMqttService(ecoFlowHttpService, mqttSubReadService, mqttForwardService, prometheusService, taskService, cfg.EcoFlow)
	if err = ecoFlowMqttService.Init(); err != nil {
		log.Fatal().Msgf("EcoFlow service initialization failed: %v", err)
	}

	ecoFlowMqttTask := service.NewEcoFlowMqttTask(ecoFlowMqttService, mqttForwardService, prometheusService, taskService, cfg.EcoFlow)
	mqttSubWriteService := service.NewMqttSubscriptionWriteService(mqttSubReadService, ecoFlowMqttTask, mqttSubRepo)
	deviceService := service.NewDeviceService(mqttSubReadService, ecoFlowMqttTask, deviceRepo)

	collectorService := service.NewCollectorService(ecoFlowHttpService, mqttForwardService, taskService, prometheusService, collectorRepo)
	if err = collectorService.Init(); err != nil {
		log.Fatal().Msgf("Collector service initialization failed: %v", err)
	}

	prometheusTask := service.NewPrometheusTask(cfg.Prometheus, ecoFlowMqttService, mqttForwardService, prometheusService, taskService)
	if err = prometheusTask.Init(); err != nil {
		log.Fatal().Msgf("Task prometheus task initialization failed: %v", err)
	}
	taskService.Start()

	// handlers init
	infoHandler := handler.NewInfoHandler(cfg.App)
	healthHandler := handler.NewHealthHandler()
	deviceHandler := handler.NewDeviceHandler(deviceService)
	mqttSubHandler := handler.NewMqttSubscriptionHandler(mqttSubReadService, mqttSubWriteService)
	collectorHandler := handler.NewCollectorHandler(collectorService)
	ecoFlowHandler := handler.NewEcoFlowHandler(ecoFlowHttpService, ecoFlowMqttService)

	apiPublicGroup := appRouter.Group(fmt.Sprintf("%s/api/v1", cfg.Server.BasePath))
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
		authMethodHandler = func(c *gin.Context) {
			return
		}
	} else {
		log.Fatal().Msgf("No valid auth mode found")
	}

	apiAuthGroup := appRouter.Group(fmt.Sprintf("%sapi/v1", cfg.Server.BasePath), authMethodHandler)

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

	apiAuthGroup.GET("/ecoflow/status", ecoFlowHandler.BrokerStatus)
	apiAuthGroup.GET("/ecoflow/devices", ecoFlowHandler.Devices)
	apiAuthGroup.POST("/ecoflow/devices/:sn", ecoFlowHandler.Parameters)
	apiAuthGroup.GET("/ecoflow/devices/:sn", ecoFlowHandler.ParametersAll)
	apiAuthGroup.GET("/ecoflow/devices/:sn/history", ecoFlowHandler.History)
	apiAuthGroup.GET("/ecoflow/devices/:sn/batteries", ecoFlowHandler.Batteries)

	// start servers (run in separate goroutines)
	appSrv := s.newServer(appRouter, fmt.Sprintf("%s:%d", cfg.Server.Listen, cfg.Server.Port))
	prometheusSrv := s.newServer(promRouter, fmt.Sprintf("%s:%d", cfg.Prometheus.Listen, cfg.Prometheus.Port))

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

	log.Info().Msgf("Shutting down...")

	timeoutCtx, timeoutCancel := context.WithTimeout(s.ctx, cfg.Server.Timeout)
	defer timeoutCancel()

	shutdownDone := make(chan struct{})
	go func() {
		taskService.Stop()
		ecoFlowMqttService.Disconnect()
		mqttForwardService.Disconnect()
		s.stopServer(s.ctx, appSrv)
		s.stopServer(s.ctx, prometheusSrv)
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
		log.Info().Msgf("Exited")
	case <-timeoutCtx.Done():
		log.Info().Msgf("Shutdown timeout of '%v' expired, exiting forcefully...", cfg.Server.Timeout)
		os.Exit(1)
	}
}

func (s *Server) newServer(r *gin.Engine, address string) *http.Server {
	if r == nil || address == "" {
		log.Fatal().Msgf("Failed to create server, engine or address is nil")
		return nil
	}

	return &http.Server{
		Addr:    address,
		Handler: r,
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
			log.Fatal().Msgf("Server cannot be started: %v", e)
		}
	}()
}

func (s *Server) stopServer(ctx context.Context, h *http.Server) {
	if h == nil {
		return
	}

	if err := h.Shutdown(ctx); err != nil {
		log.Fatal().Msgf("Shutdown failed, exited directly: %v", err)
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
