package dispatchsim

import (
	"fmt"
	"time"
)

type Simulation struct {
	Environments []*Environment
	//Environments2 map[int]*Environment // TODO: IMPT! CONVERT ARRAY OF POINTERS TO MAP.
	MasterSpeed time.Duration
	Recieve     chan string
	Send        chan string
	OrderQueue  chan string
}

func SetupSimulation() Simulation {
	return Simulation{
		Environments: make([]*Environment, 0),
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

	go OrderRetriever(s)
	go OrderDistributor(s)

	for {
		select {
		case recieveCommand := <-s.Recieve:
			command := stringToArrayInt(recieveCommand)
			switch command[0] {
			case 1: // generate environment
				fmt.Printf("[Simulator]Generate Environment %d \n", environmentId)
				noOfDrivers = noOfDrivers + command[1]
				env := SetupEnvironment(s, environmentId, command[1], false, false) //TODO: add polygon coordinates
				s.Environments = append(s.Environments, &env)
				go env.Run(startingDriverCount)                        // run environment
				startingDriverCount = startingDriverCount + command[1] // update driver id count
				environmentId++
			}
		default:
			//fmt.Println("[Simulation] Running")
		}
	}
}

func SetupEnvironment(s *Simulation, id int, noOfDrivers int, generateDrivers bool, generateTasks bool) Environment {
	return Environment{
		S:                    s,
		Id:                   id,
		DriverAgents:         make([]DriverAgent, noOfDrivers),
		IncomingDriversQueue: make(chan DriverAgent, 1000),
		TaskQueue:            make(chan Task, 10000),
		FinishQueue:          make(chan Task, 10000),
		TotalTasks:           0,
		MasterSpeed:          100,
		Quit:                 make(chan struct{}), // for stopping dispatcher and drivers
	}
}
