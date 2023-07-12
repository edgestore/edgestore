package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/edgestore/edgestore/internal/guid"
	"github.com/edgestore/edgestore/internal/server"
	"github.com/edgestore/edgestore/master"
	"github.com/go-pg/pg/v10"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func commandServe() *cobra.Command {
	var (
		cache     string
		database  string
		logFormat string
		logLevel  string
		port      int
	)
	cmd := cobra.Command{
		Use:     "serve",
		Short:   "Start HTTP server",
		Example: ShortDescription,
		Run: func(cmd *cobra.Command, args []string) {
			var db *pg.Options
			if viper.GetString("database") != "" {
				conn, err := url.Parse(viper.GetString("database"))
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(2)
				}

				pwd, _ := conn.User.Password()
				db = &pg.Options{
					Addr:     conn.Host,
					Database: strings.Replace(conn.RequestURI(), "/", "", 1),
					User:     conn.User.Username(),
					Password: pwd,
				}
			}

			var cacheOpts *redis.Options
			if viper.GetString("cache") != "" {
				conn, err := url.Parse(viper.GetString("cache"))
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(2)
				}

				pwd, _ := conn.User.Password()
				cacheOpts = &redis.Options{
					Addr:     conn.Host,
					Password: pwd,
				}
			}

			machineID, err := guid.DefaultMachineID()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}

			cfg := master.Config{
				Cache:     cacheOpts,
				Database:  db,
				Server:    server.DefaultConfig(),
				MachineID: machineID,
			}

			cfg.Server.HTTPPort = viper.GetInt("port")
			cfg.Server.LoggerFormat = viper.GetString("log_format")
			cfg.Server.LoggerLevel = viper.GetString("log_level")

			if err := serve(cfg); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVar(&cache, "cache", "localhost:6379", "Redis address")
	viper.BindPFlag("cache", cmd.Flags().Lookup("cache"))

	cmd.Flags().StringVar(&database, "database", "", "Database connection string")
	viper.BindPFlag("database", cmd.Flags().Lookup("database"))

	cmd.Flags().StringVar(&logFormat, "log-format", "json", "Logger format")
	viper.BindPFlag("log_format", cmd.Flags().Lookup("log-format"))

	cmd.Flags().StringVar(&logLevel, "log-level", "info", "Logger level")
	viper.BindPFlag("log_level", cmd.Flags().Lookup("log-level"))

	cmd.Flags().IntVar(&port, "port", 8080, "HTTP port")
	viper.BindPFlag("port", cmd.Flags().Lookup("port"))

	return &cmd
}

func serve(cfg master.Config) error {
	svc := master.New(cfg)
	return svc.Run()
}
