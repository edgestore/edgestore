package master

import (
	"github.com/edgestore/edgestore/internal/server"
	"github.com/go-pg/pg"
	"github.com/go-redis/redis"
)

type Config struct {
	Server    server.Config
	Database  *pg.Options
	Cache     *redis.Options
	MachineID uint16
}
