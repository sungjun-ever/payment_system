package main

import (
	"order_system/internal/boostrap/api"
)

func main() {
	app := api.NewApp()
	app.Run()
}
