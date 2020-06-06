package main

import (
	"fmt"

	"github.com/harrizontal/dispatchserver/dispatchsim"
)

// This file is to check whether the road networks are setup properly in the simulation
// dispatchsim.SetupRoadNetwork will be called when the simulation start
// Command: go run github.com/harrizontal/dispatchserver/cmd/dsroadnetwork
func main() {
	fmt.Println("You are on dsroadnetwork")
	dispatchsim.SetupRoadNetwork2()
}
