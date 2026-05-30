package main

import (
	"payment_system/internal/boostrap"
)

func main() {
	app := boostrap.NewApp()
	app.Run()
}
