package main

func main() {
	makeServerLabel("DFS", "0.2.0")
	Init()
	var gmux = NewRouter()
	RegisterRoutes(gmux)
	Run(gmux)
	Done()
}
