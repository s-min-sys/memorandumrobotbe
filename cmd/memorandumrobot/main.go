package main

import (
	"github.com/s-min-sys/memorandumrobotbe/internal/config"
	"github.com/s-min-sys/memorandumrobotbe/internal/server"
	"github.com/sgostarter/i/l"
)

func main() {
	server.NewServer(config.GetConfig(), l.NewConsoleLoggerWrapper()).Wait()
}
