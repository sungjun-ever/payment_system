package main

import (
	"order_system/internal/boostrap"
)

func main() {
	app := boostrap.NewApp()
	app.Run()
}
