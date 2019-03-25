package master

import (
	"net/http"
	"time"

	"github.com/edgestore/edgestore/internal/guid"

	"github.com/edgestore/edgestore/association"

	"github.com/edgestore/edgestore/entity"
	"github.com/edgestore/edgestore/internal/eventstore"
	"github.com/edgestore/edgestore/internal/eventstore/pgstore"
	"github.com/edgestore/edgestore/internal/server"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
)

// CacheKeyPrefix is used to define caching keys.
const CacheKeyPrefix = "everstore"

type Service interface{}

type service struct {
	association *association.Service
	cache       *redis.Client
	cfg         Config
	entity      *entity.Service
	guid        *guid.Generator
	logger      logrus.FieldLogger

	run func() error
}

func New(cfg Config) *service {
	if cfg.Server.LoggerLevel != "debug" {
		gin.SetMode("release")
	}

	// Logger
	logger := server.NewLogger(cfg.Server.LoggerLevel, cfg.Server.LoggerFormat)

	// Event Store
	var store eventstore.Store
	if cfg.Database == nil {
		store = eventstore.NewInMemory(logger)
	} else {
		store = pgstore.New(cfg.Database, logger)
	}

	cache := redis.NewClient(cfg.Cache)

	// Data Store Service
	assocSvc := association.New(&association.Config{
		Cache:          cache,
		CacheKeyPrefix: CacheKeyPrefix,
		Store:          store,
		Logger:         logger,
	})

	entitySvc := entity.New(&entity.Config{
		Cache:          cache,
		CacheKeyPrefix: CacheKeyPrefix,
		Store:          store,
		Logger:         logger,
	})

	guidSvc := guid.New(guid.Settings{
		StartTime: time.Now(),
		MachineID: func() (uint16, error) { return cfg.MachineID, nil },
	})

	// Main Service
	svc := &service{
		association: assocSvc,
		cache:       cache,
		cfg:         cfg,
		entity:      entitySvc,
		guid:        guidSvc,
		logger:      logger.WithField("component", "API"),
	}

	srv := server.New(cfg.Server, logger)
	srv.HTTPServer = server.NewHTTPServer(cfg.Server, svc.HTTPHandler())
	svc.run = srv.Run

	return svc
}

func (s *service) Run() error {
	s.logger.Info("Edgestore: Starting Master")

	if s.cache != nil {
		if _, err := s.cache.Ping().Result(); err != nil {
			s.logger.Errorf("unable to connect to Redis: %v", err)
		} else {
			s.logger.Infof("Connected to Redis at %v", s.cfg.Cache.Addr)
		}
	}

	return s.run()
}

func (s *service) Shutdown() {
	s.logger.Info("Edgestore: Stopping Master")

	if s.cache != nil {
		if _, err := s.cache.Shutdown().Result(); err != nil {
			s.logger.Error(err)
		}
	}
}

func (s *service) RootHandler(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"message": "Edgestore: Distributed Data Store (Master)",
	})
}
