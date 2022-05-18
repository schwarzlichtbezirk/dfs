package main

func main() {
	Init()
	var gmux = NewRouter()
	RegisterRoutes(gmux)
	Run(gmux)
	Done()
}
