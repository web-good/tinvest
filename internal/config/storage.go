package config

import "net"

type Storage struct {
	DbHost string `config:"DB_HOST"`
	DbPort string `config:"DB_PORT"`
	DbName string `config:"DB_NAME"`
	DbPass string `config:"DB_PASS"`
	DbUser string `config:"DB_USER"`
}

func (s *Storage) HostPort() string {
	return net.JoinHostPort(s.DbHost, s.DbPort)
}

func (s *Storage) Dsn() string {
	return "user=" + s.DbUser + " host=" + s.DbHost + " dbname=" + s.DbName + " sslmode=disable password=" + s.DbPass
}
