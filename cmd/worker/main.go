package main

import "order_system/internal/boostrap/worker"

func main() {
	app := worker.NewApp()
	app.Run()
}
