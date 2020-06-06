package main

import (
	"fmt"
	"sync"

	"github.com/harrizontal/dispatchserver/dispatchsim"
)

func main() {
	fmt.Println("Starting sim...")
	var wg sync.WaitGroup

	env := dispatchsim.SetupEnvironment(3, 10, &wg)
	fmt.Println("[App] Run Simulation start")
	wg.Add(1)
	go env.Run()
	wg.Wait()
	env.Stats()
	fmt.Println("[App] Run Simulation end")
}
