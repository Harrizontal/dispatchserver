package dispatchsim

import (
	"encoding/json"
	"fmt"
	"time"
)

type Simulation struct {
	Environments  map[int]*Environment // TODO: IMPT! CONVERT ARRAY OF POINTERS TO MAP.
	MasterSpeed   time.Duration
	Recieve       chan string
	Send          chan string
	OrderQueue    chan Order
	GeoJSONUpdate bool
}

func SetupSimulation() Simulation {
	return Simulation{
		Environments:  make(map[int]*Environment),
		MasterSpeed:   200,
		Recieve:       make(chan string, 10000),
		Send:          make(chan string, 10000),
		OrderQueue:    make(chan Order, 1000),
		GeoJSONUpdate: true, // set true to update mapbox
	}
}

func (s *Simulation) Run() {
	var environmentId = 1 // starting id
	var noOfDrivers = 0
	var startingDriverCount = 1 // starting id of driver

	var k = false
	tick := time.Tick((100) * time.Millisecond)
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
					s.Send <- "[Simulator]Cannot intailize order distributor"
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
				eId := int(command.([]interface{})[2].(float64)) // get environmentid from the message
				dId := int(command.([]interface{})[3].(float64)) // get driverId from the message
				switch commandTypeLevelTwo {
				case 0:
					//fmt.Printf("[Simulator]Recieve random point from client and send to driver\n")
					sendRandomPointDriver(command, s, eId, dId)
				case 1:
					//fmt.Printf("[Simulator]Recieve Intialization from client and send to driver\n")
					// random start location, random destination, waypoint
					sendIntializationDriver(command, s, eId, dId)
				case 2: // waypoint
					sendWaypointsDriver(command, s, eId, dId)
				case 3: // generate node
					sendGenerateResultDriver(command, s, eId, dId)
				case 4: // driver move
					sendMoveResultDriver(command, s, eId, dId)
				case 5: // random destination and waypoint
					sendRandomDestinationWaypoint(command, s, eId, dId)
				}
			}
		default:
			//fmt.Println("[Simulation] Running")
		}
		// Send GEOJSON every X miliseconds
		select {
		case <-tick:
			if s.GeoJSONUpdate == true {
				SendGeoJSON(s)
			}
		default:

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
		Waypoint:          ConvertToArrayLatLng(command.([]interface{})[4].([]interface{})),
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
	location := ConvertToLatLng(command.([]interface{})[4].([]interface{}))
	//fmt.Printf("%T %[1]v\n", success)
	message := Message{
		CommandType:       2,
		CommandSecondType: 4,
		LocationArrived:   location,
		Success:           true, //TODO!!! remove this!
	}

	s.Environments[eId].DriverAgents[dId].Recieve <- message
}

func sendRandomDestinationWaypoint(command interface{}, s *Simulation, eId int, dId int) {
	// fmt.Printf("%T %[1]v \n", command.([]interface{})[6].([]interface{}))
	startLocation := ConvertToLatLng(command.([]interface{})[4].([]interface{}))
	destLocation := ConvertToLatLng(command.([]interface{})[5].([]interface{}))
	waypoints := ConvertToArrayLatLng(command.([]interface{})[6].([]interface{}))

	message := Message{
		CommandType:       2,
		CommandSecondType: 5,
		StartDestinationWaypoint: StartDestinationWaypoint{
			StartLocation:       startLocation,
			DestinationLocation: destLocation,
			Waypoint:            waypoints,
		},
	}

	s.Environments[eId].DriverAgents[dId].Recieve <- message
}

func SendGeoJSON(s *Simulation) {
	if len(s.Environments) > 0 {
		geojson := &GeoJSONFormat{
			Type:     "FeatureCollection",
			Features: make([]Feature, 0),
		}
		//fmt.Printf("[Simulator]Sending geojson\n")
		for _, v := range s.Environments {
			// add drivers to geojson
			for _, v2 := range v.DriverAgents {
				//fmt.Printf("[Simulator] Driver %d\n", v2.Id)
				feature := Feature{
					Type: "Feature",
					Geometry: Geometry{
						Type:        "Point",
						Coordinates: latlngToArrayFloat(v2.CurrentLocation),
					},
					Properties: Properties{
						Type: "Driver",
						Information: DriverFormat{
							Id:          v2.Id,
							Status:      v2.Status,
							CurrentTask: v2.CurrentTask.Id,
						},
					},
				}
				geojson.Features = append(geojson.Features, feature)
			}
			// add tasks to geojson
			for _, v3 := range v.Tasks {
				if (v3.TaskEnded == time.Time{}) {
					feature := Feature{
						Type: "Feature",
						Geometry: Geometry{
							Type:        "Point",
							Coordinates: latlngToArrayFloat(v3.StartPosition),
						},
						Properties: Properties{
							Type: "Task",
							Information: TaskFormat{
								Id:            v3.Id,
								StartPosition: latlngToArrayFloat(v3.StartPosition),
								EndPosition:   latlngToArrayFloat(v3.EndPosition),
								WaitStart:     v3.WaitStart,
								WaitEnd:       v3.WaitEnd,
								TaskEnd:       v3.TaskEnded,
							},
						},
					}
					geojson.Features = append(geojson.Features, feature)
				}

			}
		}

		e, err := json.Marshal(geojson)
		if err != nil {
			fmt.Println(err)
		}
		s.Send <- string(e)
	}

}
