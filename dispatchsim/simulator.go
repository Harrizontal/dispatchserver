package dispatchsim

import (
	"fmt"
	"time"
)

type Simulation struct {
	Environments map[int]*Environment // TODO: IMPT! CONVERT ARRAY OF POINTERS TO MAP.
	MasterSpeed  time.Duration
	Recieve      chan string
	Send         chan string
	OrderQueue   chan string
}

func SetupSimulation() Simulation {
	return Simulation{
		Environments: make(map[int]*Environment),
		MasterSpeed:  200,
		Recieve:      make(chan string, 10000),
		Send:         make(chan string, 10000),
		OrderQueue:   make(chan string, 1000),
	}
}

func (s *Simulation) Run() {
	var environmentId = 1 // starting id
	var noOfDrivers = 0
	var startingDriverCount = 1 // starting id of driver

	var k = false

	for {
		select {
		case recieveCommand := <-s.Recieve:
			command := stringToArrayInt(recieveCommand)
			switch command[0] {
			case 0:
				// initializing order retriever, and order distributor
				if len(s.Environments) > 0 && k == false {
					k = true
					go OrderRetriever(s)
					go OrderDistributor(s)
				} else {
					s.Send <- "Cannot intailize order distributor"
				}
			case 1: // generate environment
				fmt.Printf("[Simulator]Generate Environment %d \n", environmentId)
				noOfDrivers = noOfDrivers + command[1]
				env := SetupEnvironment(s, environmentId, command[1], false, false) //TODO: add polygon coordinates
				s.Environments[environmentId] = &env
				go env.Run(startingDriverCount)                        // run environment
				startingDriverCount = startingDriverCount + command[1] // update driver id count
				environmentId++
			}
		default:
			//fmt.Println("[Simulation] Running")
		}
	}
}
