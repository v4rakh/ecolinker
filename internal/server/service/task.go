package service

import (
	"fmt"
	"git.myservermanager.com/varakh/ecolinker/internal/app"
	"git.myservermanager.com/varakh/ecolinker/internal/server/config"
	"git.myservermanager.com/varakh/ecolinker/internal/server/service_error"
	"github.com/go-co-op/gocron-redis-lock/v2"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"time"
)

type TaskService struct {
	lockService LockService
	appConfig   *config.App
	lockConfig  *config.Lock
	scheduler   gocron.Scheduler
}

// NewTaskService constructs the service, ensuring bootstrapped connection to potential redis works
func NewTaskService(l LockService, ac *config.App, lc *config.Lock) (*TaskService, error) {
	var err error
	var location *time.Location
	if location, err = time.LoadLocation(ac.TimeZone); err != nil {
		return nil, fmt.Errorf("could not initialize correct timezone for scheduler: %s", err)
	}

	// global job options
	singletonModeOption := gocron.WithSingletonMode(gocron.LimitModeReschedule)
	errorEventListener := gocron.AfterJobRunsWithError(func(jobID uuid.UUID, jobName string, err error) {
		log.Error().Msgf("Job '%s' (%v) had a panic %v", jobName, jobID, err)
	})
	successEventListener := gocron.AfterJobRuns(func(jobID uuid.UUID, jobName string) {
		log.Debug().Msgf("Job '%s' (%v) finished", jobName, jobID)
	})
	beforeEventListener := gocron.BeforeJobRuns(func(jobID uuid.UUID, jobName string) {
		log.Debug().Msgf("Job '%s' (%v) starts", jobName, jobID)
	})
	eventListenerOption := gocron.WithEventListeners(beforeEventListener, successEventListener, errorEventListener)

	// global scheduler options
	schedulerOptions := []gocron.SchedulerOption{gocron.WithLocation(location), gocron.WithGlobalJobOptions(singletonModeOption, eventListenerOption)}

	if lc.RedisEnabled {
		log.Info().Msg("Initializing REDIS task service")

		var c *redis.Client
		if c, err = config.NewRedisClient(fmt.Sprintf("%s-task", app.Name), lc.RedisUrl); err != nil {
			return nil, fmt.Errorf("task service: cannot initialize REDIS client: %w", err)
		}

		var locker gocron.Locker
		if locker, err = redislock.NewRedisLocker(c, redislock.WithTries(lc.RedisTaskTries), redislock.WithExpiry(lc.RedisTaskLockAtMost), redislock.WithRetryDelay(lc.RedisTaskRetryDelay)); err != nil {
			return nil, fmt.Errorf("task service: cannot set up REDIS locker: %w", err)
		}

		schedulerOptions = append(schedulerOptions, gocron.WithDistributedLocker(locker))
	}

	scheduler, _ := gocron.NewScheduler(schedulerOptions...)

	return &TaskService{
		lockService: l,
		appConfig:   ac,
		lockConfig:  lc,
		scheduler:   scheduler,
	}, nil
}

// Start starts the scheduler, should be called after Init
func (s *TaskService) Start() {
	s.scheduler.Start()
	log.Info().Msgf("Started %d periodic tasks", len(s.scheduler.Jobs()))
}

// Stop stops the service and shuts down the scheduler
func (s *TaskService) Stop() {
	log.Info().Msgf("Stopping %d periodic tasks...", len(s.scheduler.Jobs()))
	if err := s.scheduler.StopJobs(); err != nil {
		log.Warn().Msgf("Cannot stop periodic tasks. Reason: %v", err)
	}
	if err := s.scheduler.Shutdown(); err != nil {
		log.Warn().Msgf("Cannot shut down scheduler. Reason: %v", err)
	}
	log.Info().Msg("Stopped all periodic tasks")
}

// EnqueueOnce enqueues a new job once for execution, convenience method for gocron.WithLimitedRuns, see https://github.com/go-co-op/gocron/issues/709
func (s *TaskService) EnqueueOnce(job gocron.JobDefinition, task gocron.Task, name string, options ...gocron.JobOption) (gocron.Job, error) {
	jobOptions := []gocron.JobOption{gocron.WithLimitedRuns(1)}
	jobOptions = append(jobOptions, options...)
	return s.Enqueue(job, task, name, jobOptions...)
}

// Enqueue enqueues a new job
func (s *TaskService) Enqueue(job gocron.JobDefinition, task gocron.Task, name string, options ...gocron.JobOption) (gocron.Job, error) {
	if name == "" {
		return nil, service_error.ErrValidationNotBlank
	}
	jobOptions := []gocron.JobOption{gocron.WithName(name)}
	jobOptions = append(jobOptions, options...)
	return s.scheduler.NewJob(job, task, jobOptions...)
}

// Cancel cancels a job by ID
func (s *TaskService) Cancel(id uuid.UUID) error {
	log.Debug().Msgf("Removing by ID '%v'", id)
	return s.scheduler.RemoveJob(id)
}

// CancelByTag cancels a job by tags
func (s *TaskService) CancelByTag(tags ...string) {
	log.Debug().Msgf("Removing by tags '%v'", tags)
	s.scheduler.RemoveByTags(tags...)
}
