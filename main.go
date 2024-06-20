package main

import (
	"github.com/Suhaibinator/SuhaibMessageQueue/config"
	"github.com/Suhaibinator/SuhaibMessageQueue/server"
)

func main() {

	server := server.NewServer(config.Port)
	server.Start()

}
