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
			command := stringToArrayString(recieveCommand)
			//fmt.Println(command)
			commandType := command.([]interface{})[0].(float64)
			switch commandType {
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
				inputNoOfDrivers := int(command.([]interface{})[1].(float64))
				noOfDrivers = noOfDrivers + inputNoOfDrivers
				env := SetupEnvironment(s, environmentId, inputNoOfDrivers, false, false) //TODO: add polygon coordinates
				s.Environments[environmentId] = &env
				go env.Run(startingDriverCount)                              // run environment
				startingDriverCount = startingDriverCount + inputNoOfDrivers // update driver id count
				environmentId++
			case 2: // drivers
				commandTypeLevelTwo := command.([]interface{})[1].(float64)
				eId := int(command.([]interface{})[2].(float64))
				dId := int(command.([]interface{})[3].(float64))
				switch commandTypeLevelTwo {
				case 0:
					//fmt.Printf("[Simulator]Recieve random point from client and send to driver\n")
					sendRandomPointDriver(command, s, eId, dId)
				case 1:
					//fmt.Printf("[Simulator]Recieve Intialization from client and send to driver\n")
					sendIntializationDriver(command, s, eId, dId)
				case 2: // waypoint
					sendWaypointsDriver(command, s, eId, dId)
				case 3: // generate node
					sendGenerateResultDriver(command, s, eId, dId)
				case 4: // driver move
					sendMoveResultDriver(command, s, eId, dId)
				}
			}
		default:
			//fmt.Println("[Simulation] Running")
		}
	}
}

// func (s *Environment) genereateEnvironment(command interface{}, environmentId *int, noOfDrivers *int, startingDriverCount *int) {
// 	fmt.Printf("[Simulator]Generate Environment %d \n", environmentId)
// 	inputNoOfDrivers := int(command.([]interface{})[1].(float64))
// 	noOfDrivers = noOfDrivers + inputNoOfDrivers
// 	env := SetupEnvironment(s, environmentId, inputNoOfDrivers, false, false) //TODO: add polygon coordinates
// 	s.Environments[environmentId] = &env
// 	go env.Run(startingDriverCount)                              // run environment
// 	startingDriverCount = startingDriverCount + inputNoOfDrivers // update driver id count
// 	environmentId++
// }

// e.g 2,0,{environmentId},{DriverId}
func sendRandomPointDriver(command interface{}, s *Simulation, eId int, dId int) {
	latLng := ConvertToLatLng(command.([]interface{})[4].([]interface{}))
	s.Environments[eId].DriverAgents[dId].Recieve <- Message{CommandType: 2, CommandSecondType: 0, LatLng: latLng}
}

func sendIntializationDriver(command interface{}, s *Simulation, eId int, dId int) {
	// fmt.Printf("%T %[1]v \n", command.([]interface{})[6].([]interface{}))
	startLocation := ConvertToLatLng(command.([]interface{})[4].([]interface{}))
	destLocation := ConvertToLatLng(command.([]interface{})[5].([]interface{}))
	waypoints := ConvertToArrayLatLng(command.([]interface{})[6].([]interface{}))

	message := Message{
		CommandType:       2,
		CommandSecondType: 1,
		StartDestinationWaypoint: StartDestinationWaypoint{
			StartLocation:       startLocation,
			DestinationLocation: destLocation,
			Waypoint:            waypoints,
		},
	}

	s.Environments[eId].DriverAgents[dId].Recieve <- message
}

func sendWaypointsDriver(command interface{}, s *Simulation, eId int, dId int) {

	message := Message{
		CommandType:       2,
		CommandSecondType: 2,
		Waypoint:          ConvertToArrayLatLng(command.([]interface{})[3].([]interface{})),
	}

	s.Environments[eId].DriverAgents[dId].Recieve <- message
}

func sendGenerateResultDriver(command interface{}, s *Simulation, eId int, dId int) {
	success := command.([]interface{})[4].(bool)
	//fmt.Printf("%T %[1]v\n", success)
	message := Message{
		CommandType:       2,
		CommandSecondType: 3,
		Success:           success,
	}

	s.Environments[eId].DriverAgents[dId].Recieve <- message
}

func sendMoveResultDriver(command interface{}, s *Simulation, eId int, dId int) {
	success := command.([]interface{})[4].(bool)
	//fmt.Printf("%T %[1]v\n", success)
	message := Message{
		CommandType:       2,
		CommandSecondType: 4,
		Success:           success,
	}

	s.Environments[eId].DriverAgents[dId].Recieve <- message
}
