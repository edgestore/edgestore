package master

import (
	"github.com/edgestore/edgestore/internal/server"
	"github.com/go-pg/pg"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Server    server.Config
	Database  *pg.Options
	Cache     *redis.Options
	MachineID uint16
}
