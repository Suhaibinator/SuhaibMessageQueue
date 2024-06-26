package main

import (
	"github.com/Suhaibinator/SuhaibMessageQueue/config"
	"github.com/Suhaibinator/SuhaibMessageQueue/server"
)

func main() {

	server := server.NewServer(config.Port, config.DBPath)
	server.Start()

}
