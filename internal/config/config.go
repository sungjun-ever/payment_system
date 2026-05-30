package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort     string
	MysqlDBHost string
	MysqlDBPort string
	MysqlDBUser string
	MysqlDBPass string
	MysqlDBName string
	RedisHost   string
	RedisPort   string
	JwtSecret   string
}

func Load() *Config {
	err := godotenv.Load()

	if err != nil {
		log.Fatalf("ENV 로딩 실패: %s", err)
	}

	appPort := os.Getenv("APP_PORT")
	mysqlDBHost := os.Getenv("MYSQL_DB_HOST")
	mysqlDBPort := os.Getenv("MYSQL_DB_PORT")
	mysqlDBUser := os.Getenv("MYSQL_DB_USER")
	mysqlDBPass := os.Getenv("MYSQL_DB_PASS")
	mysqlDBName := os.Getenv("MYSQL_DB_NAME")
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	jwtSecret := os.Getenv("JWT_SECRET")

	return &Config{
		AppPort:     appPort,
		MysqlDBHost: mysqlDBHost,
		MysqlDBPort: mysqlDBPort,
		MysqlDBUser: mysqlDBUser,
		MysqlDBPass: mysqlDBPass,
		MysqlDBName: mysqlDBName,
		RedisHost:   redisHost,
		RedisPort:   redisPort,
		JwtSecret:   jwtSecret,
	}
}
