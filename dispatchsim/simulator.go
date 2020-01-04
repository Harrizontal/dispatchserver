package dispatchsim

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type Simulation struct {
	Environments     map[int]*Environment
	DriverAgents     map[int]*DriverAgent
	DriverAgentMutex sync.RWMutex
	OM               *OrderManager
	MasterSpeed      time.Duration
	Recieve          chan string // recieve from websocket
	Send             chan string // send to websocket
	OrderQueue       chan Order
	UpdateMap        bool
	UpdateMapSpeed   time.Duration
}

func SetupSimulation() Simulation {
	return Simulation{
		Environments:     make(map[int]*Environment),
		DriverAgents:     make(map[int]*DriverAgent),
		DriverAgentMutex: sync.RWMutex{},
		MasterSpeed:      200,
		Recieve:          make(chan string, 10000),
		Send:             make(chan string, 10000),
		OrderQueue:       make(chan Order, 1000),
		UpdateMap:        true, // set true to update mapbox
		UpdateMapSpeed:   1000, // update speed to mapbox
	}
}

func (s *Simulation) Run() {
	var environmentId = 1 // starting id
	var noOfDrivers = 0
	var startingDriverCount = 1 // starting id of driver

	//var k = false
	tick := time.Tick((s.UpdateMapSpeed) * time.Millisecond)
	for {
		select {
		case recieveCommand := <-s.Recieve:
			command := stringToArrayString(recieveCommand)
			fmt.Printf("[Sim]%v\n", recieveCommand)
			commandType := command.([]interface{})[0].(float64)
			switch commandType {
			case 0: // pause

			K:
				for {
					select {
					case recieveCommand := <-s.Recieve:
						command := stringToArrayString(recieveCommand)
						commandType := command.([]interface{})[0].(float64)
						//fmt.Printf("[Simulator]Receive %v \n", command)
						switch commandType {
						case 0:
							fmt.Printf("[Simulator]Play\n")
							break K
						default:
							//fmt.Printf("[Simulator]Recieve something when pause %v\n", commandType)
							s.Recieve <- recieveCommand
						}
					default:
						//fmt.Printf("[Simulator]On Pause\n")
					}
				}

				fmt.Printf("[Simulator]End of case 0\n")
			case 1: // generate environment
				fmt.Printf("[Simulator]Generate Environment %d \n", environmentId)
				inputNoOfDrivers := int(command.([]interface{})[1].(float64))
				latLngs := command.([]interface{})[2].([]interface{})
				noOfDrivers = noOfDrivers + inputNoOfDrivers
				env := SetupEnvironment(s, environmentId, inputNoOfDrivers, false, false, ConvertToArrayLatLng(latLngs))
				s.Environments[environmentId] = &env
				CreateMultipleDrivers(startingDriverCount, inputNoOfDrivers, &env, s)
				go env.Run()                                                 // run environment
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
			case 3: // order distributor
				commandTypeLevelTwo := command.([]interface{})[1].(float64)
				switch commandTypeLevelTwo {
				case 0:
					om := SetupOrderRetrieve(s)
					s.OM = &om
					go om.runOrderRetrieve()
					go om.runOrderDistribute()

					// if len(s.Environments) > 0 && k == false {
					// 	k = true
					// 	om := SetupOrderRetrieve(s)
					// 	s.OM = &om
					// 	go om.runOrderRetrieve()
					// 	go om.runOrderDistribute()
					// } else {
					// 	s.Send <- "[Simulator]Cannot intailize order distributor"
					// }
				case 1: // pickup lnglat and drop off lnglat in terms of waypoint
					fmt.Printf("accessing 3,1\n")
					sendCorrectedLocation(command, s)
				case 2:
					om := SetupOrderRetrieve(s)
					s.OM = &om
					go s.OM.getCoord()
				}
				// initializing order retriever, and order distributor

			}
		}

		// Send GEOJSON every X miliseconds
		select {
		case <-tick:
			if s.UpdateMap {
				SendGeoJSON2(s)
			}
		default:

		}
	}
	fmt.Println("[Simulation]Ended")
}

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

	s.DriverAgents[dId].Recieve <- message
}

func sendWaypointsDriver(command interface{}, s *Simulation, eId int, dId int) {

	message := Message{
		CommandType:       2,
		CommandSecondType: 2,
		Waypoint:          ConvertToArrayLatLng(command.([]interface{})[4].([]interface{})),
	}

	s.DriverAgents[dId].Recieve <- message
}

func sendGenerateResultDriver(command interface{}, s *Simulation, eId int, dId int) {
	success := command.([]interface{})[4].(bool)
	//fmt.Printf("%T %[1]v\n", success)
	message := Message{
		CommandType:       2,
		CommandSecondType: 3,
		Success:           success,
	}

	s.DriverAgents[dId].Recieve <- message
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

	s.DriverAgents[dId].Recieve <- message
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

	s.DriverAgents[dId].Recieve <- message
}

func sendCorrectedLocation(command interface{}, s *Simulation) {
	pickupLocation := ConvertLngLatToLatLng(command.([]interface{})[2].([]interface{}))
	dropoffLocation := ConvertLngLatToLatLng(command.([]interface{})[3].([]interface{}))

	r := RecieveFormat{
		Command:       3,
		CommandSecond: 1,
		Data: CorrectedLocation{
			StartCoordinate: pickupLocation,
			EndCoordinate:   dropoffLocation,
		},
	}

	s.OM.Recieve <- r
}

func SendGeoJSON(s *Simulation) {
	if len(s.Environments) > 0 {
		geojson := &GeoJSONFormat{
			Type:     "FeatureCollection",
			Features: make([]Feature, 0),
		}
		for _, v := range s.Environments {
			//TODO: Separate polygon into another sendgeojson function
			feature := Feature{
				Type: "Feature",
				Geometry: Geometry2{
					Type:        "Polygon",
					Coordinates: twoLatLngtoArrayFloat(v.PolygonLatLng),
				},
				Properties: Properties{},
			}
			geojson.Features = append(geojson.Features, feature)
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
							Id:            v2.Id,
							EnvironmentId: v2.E.Id,
							Status:        v2.Status,
							CurrentTask:   v2.CurrentTask.Id,
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
							Coordinates: latlngToArrayFloat(v3.PickUpLocation),
						},
						Properties: Properties{
							Type: "Task",
							Information: TaskFormat{
								Id:            v3.Id,
								EnvironmentId: v3.EnvironmentId,
								StartPosition: latlngToArrayFloat(v3.PickUpLocation),
								EndPosition:   latlngToArrayFloat(v3.DropOffLocation),
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

func SendGeoJSON2(s *Simulation) {

	geojson := &GeoJSONFormat{
		Type:     "FeatureCollection",
		Features: make([]Feature, 0),
	}

	for _, v := range s.Environments {
		//TODO: Separate polygon into another sendgeojson function
		feature := Feature{
			Type: "Feature",
			Geometry: Geometry2{
				Type:        "Polygon",
				Coordinates: twoLatLngtoArrayFloat(v.PolygonLatLng),
			},
			Properties: Properties{},
		}
		geojson.Features = append(geojson.Features, feature)

		// add tasks to geojson
		for _, v3 := range v.Tasks {
			if (v3.TaskEnded == time.Time{}) {
				feature := Feature{
					Type: "Feature",
					Geometry: Geometry{
						Type:        "Point",
						Coordinates: latlngToArrayFloat(v3.PickUpLocation),
					},
					Properties: Properties{
						Type: "Task",
						Information: TaskFormat{
							Id:            v3.Id,
							EnvironmentId: v3.EnvironmentId,
							StartPosition: latlngToArrayFloat(v3.PickUpLocation),
							EndPosition:   latlngToArrayFloat(v3.DropOffLocation),
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

	for _, v := range s.DriverAgents {
		feature := Feature{
			Type: "Feature",
			Geometry: Geometry{
				Type:        "Point",
				Coordinates: latlngToArrayFloat(v.CurrentLocation),
			},
			Properties: Properties{
				Type: "Driver",
				Information: DriverFormat{
					Id:            v.Id,
					EnvironmentId: v.E.Id,
					Status:        v.Status,
					CurrentTask:   v.CurrentTask.Id,
				},
			},
		}
		geojson.Features = append(geojson.Features, feature)
	}

	e, err := json.Marshal(geojson)
	if err != nil {
		fmt.Println(err)
	}
	s.Send <- string(e)

}
