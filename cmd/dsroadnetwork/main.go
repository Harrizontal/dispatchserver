package main

import (
	"fmt"

	"github.com/harrizontal/dispatchserver/dispatchsim"
)

func main() {
	fmt.Println("You are on dsroadnetwork")
	dispatchsim.SetupRoadNetwork2()
	// rn.ExampleQuadtree_Find()
}
