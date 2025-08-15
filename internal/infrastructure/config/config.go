package config

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Env         string
	GRPCServer  GRPCServer
	Database    Database
	UserService UserService
	Prometheus  Prometheus
	Redis       Redis
}

type GRPCServer struct {
	Address string
	Port    int
}

type Database struct {
	Username       string
	Password       string
	Host           string
	Port           string
	DbName         string
	MigrationsPath string
}

type UserService struct {
	Address string
	Port    int
}

type Prometheus struct {
	Address string
	Port    int
}

type Redis struct {
	Address  string
	Port     int
	Password string
	DB       int
	PoolSize int
}

func MustLoad() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")

	viper.SetDefault("env", "dev")

	viper.SetDefault("grpc_server.address", "0.0.0.0")
	viper.SetDefault("grpc_server.port", 50053)

	viper.SetDefault("database.username", "postgres")
	viper.SetDefault("database.password", "admin")
	viper.SetDefault("database.host", "post-db")
	viper.SetDefault("database.port", "5434")
	viper.SetDefault("database.db_name", "postservice")
	viper.SetDefault("database.migrations_path", "migrations")

	viper.SetDefault("user_service.address", "user-service")
	viper.SetDefault("user_service.port", 50051)

	viper.SetDefault("prometheus.address", "0.0.0.0")
	viper.SetDefault("prometheus.port", 9103)

	viper.SetDefault("redis.address", "redis")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.pool_size", 10)

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Error reading config file: %s", err)
		os.Exit(1)
	}

	config := &Config{
		Env: viper.GetString("env"),
		GRPCServer: GRPCServer{
			Address: viper.GetString("grpc_server.address"),
			Port:    viper.GetInt("grpc_server.port"),
		},
		Database: Database{
			Username:       viper.GetString("database.username"),
			Password:       viper.GetString("database.password"),
			Host:           viper.GetString("database.host"),
			Port:           viper.GetString("database.port"),
			DbName:         viper.GetString("database.db_name"),
			MigrationsPath: viper.GetString("database.migrations_path"),
		},
		UserService: UserService{
			Address: viper.GetString("user_service.address"),
			Port:    viper.GetInt("user_service.port"),
		},
		Prometheus: Prometheus{
			Address: viper.GetString("prometheus.address"),
			Port:    viper.GetInt("prometheus.port"),
		},
		Redis: Redis{
			Address:  viper.GetString("redis.address"),
			Port:     viper.GetInt("redis.port"),
			Password: viper.GetString("redis.password"),
			DB:       viper.GetInt("redis.db"),
			PoolSize: viper.GetInt("redis.pool_size"),
		},
	}

	return config
}
