package config

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"interview-guide-go/shared/errmsg"
)

type ServerConfig struct {
	// server 配置
	ServerHost string
	// server 端口
	ServerPort int
	// server 读超时时间
	ServerReadTimeoutSec int
}

// 验证 server 配置
func validateServerConfig() (*ServerConfig, error) {
	// server 配置
	serverHost := strings.TrimSpace(os.Getenv("SERVER_HOST"))
	if serverHost == "" {
		return nil, errors.New(errmsg.ConfigServerHostRequired)
	}
	// server 端口
	serverPort, err := strconv.Atoi(os.Getenv("SERVER_PORT"))
	if err != nil || serverPort <= 0 {
		return nil, errors.New(errmsg.ConfigServerPortInvalid)
	}
	// server 读超时时间
	serverReadTimeoutSec, err := strconv.Atoi(os.Getenv("SERVER_READ_TIMEOUT_SECONDS"))
	if err != nil || serverReadTimeoutSec <= 0 {
		return nil, errors.New(errmsg.ConfigServerReadTimeoutInvalid)
	}
	return &ServerConfig{
		ServerHost:           serverHost,
		ServerPort:           serverPort,
		ServerReadTimeoutSec: serverReadTimeoutSec,
	}, nil
}
