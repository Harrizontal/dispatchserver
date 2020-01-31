package dispatchsim

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

type Simulation struct {
	isRunning            bool
	Environments         map[int]*Environment
	DriverAgents         map[int]*DriverAgent
	DriverAgentMutex     sync.RWMutex
	OM                   *OrderManager
	MasterSpeed          time.Duration
	Recieve              chan string // recieve from websocket
	Send                 chan string // send to websocket
	OrderQueue           chan Order
	UpdateMap            bool          // settings
	UpdateMapSpeed       time.Duration // settings
	DispatcherSpeed      time.Duration
	UpdateStatsSpeed     time.Duration
	Ticker               <-chan time.Time
	SimulationTime       time.Time
	TaskParameters       TaskParametersFormat
	DispatcherParameters DispatcherParametersFormat
}

func SetupSimulation() Simulation {

	defaultTaskParameters := TaskParametersFormat{
		TaskValueType:       "actual",
		ValuePerKM:          1,
		PeakHourRate:        1,
		ReputationGivenType: "random",
		ReputationValue:     0,
	}

	defaultDispatcherParameters := DispatcherParametersFormat{
		DispatchInterval:  5000,
		SimilarReputation: 0.5,
	}

	return Simulation{
		isRunning:            false,
		Environments:         make(map[int]*Environment),
		DriverAgents:         make(map[int]*DriverAgent),
		DriverAgentMutex:     sync.RWMutex{},
		MasterSpeed:          50,
		Recieve:              make(chan string, 10000),
		Send:                 make(chan string, 10000),
		OrderQueue:           make(chan Order, 1000),
		UpdateMap:            true, // set true to update mapbox
		UpdateMapSpeed:       100,  // update speed to mapbox
		DispatcherSpeed:      5000,
		UpdateStatsSpeed:     1000,
		Ticker:               time.Tick(50 * time.Millisecond),                 // TODO: Make adjustable
		SimulationTime:       time.Date(2020, 1, 23, 00, 30, 0, 0, time.Local), // TODO: Make adjustable
		TaskParameters:       defaultTaskParameters,
		DispatcherParameters: defaultDispatcherParameters,
	}
}

func (s *Simulation) Run() {
	var environmentId = 1 // starting id
	var noOfDrivers = 0
	var startingDriverCount = 1 // starting id of driver

	var k = false

	go s.SendMapData()
	//go s.SendStats()

	for {
		select {
		case recieveCommand := <-s.Recieve:
			command := stringToArrayString(recieveCommand)
			//fmt.Printf("[Sim]%v\n", recieveCommand)

			commandType := command.([]interface{})[0].(float64)
			switch commandType {
			case 0: // pause
				commandTypeLevelTwo := command.([]interface{})[1].(float64)
				switch commandTypeLevelTwo {
				case 0:
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
				case 1: // settings
					settings := command.([]interface{})[2]
					byteData, _ := json.Marshal(settings)
					var sf SettingsFormat
					if err := json.Unmarshal(byteData, &sf); err != nil {
						log.Fatal(err)
					}
					s.TaskParameters = sf.TaskParameters
					s.DispatcherParameters = sf.DispatcherParameters
					fmt.Printf("Task Value: %v \n", s.TaskParameters.ValuePerKM)
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
					//fmt.Printf("accessing 3,0\n")
					// om := SetupOrderRetrieve(s)
					// s.OM = &om
					// go s.StartTimer()
					// go om.runOrderRetrieve()
					// go om.runOrderDistributer()

					if len(s.Environments) > 0 && k == false {
						k = true
						go s.StartTimer()
						go s.OM.runOrderDistributer()
					} else {
						s.Send <- "[Simulator]Cannot intailize order distributor"
					}
				case 1: // pickup lnglat and drop off lnglat in terms of waypoint
					//fmt.Printf("accessing 3,1\n")
					sendCorrectedLocation(command, s)
				case 2:
					// initializing order retriever
					om := SetupOrderRetrieve(s)
					s.OM = &om
					go om.runOrderRetrieve()

				}
			}
		}
	}
	fmt.Println("[Simulation]Ended")
}

func (s *Simulation) SendMapData() {
	fmt.Printf("[Simulator]sendMapData started \n")
	tick := time.Tick((s.UpdateMapSpeed) * time.Millisecond)
	for {
		select {
		case <-tick:
			//fmt.Printf("[Simulator]Sending map data\n")
			if s.UpdateMap && s.isRunning { // send updates when there is Driver Agent available and environment placed
				SendGeoJSON(s)
				SendEnvGeoJSON(s)
				SendTaskGeoJSON(s)
			}
		default:

		}
	}
}

func (s *Simulation) SendStats() {
	fmt.Printf("[Simulator]sendStats started \n")
	tick := time.Tick((s.UpdateStatsSpeed) * time.Millisecond)
	for {
		select {
		case <-tick:
			if s.isRunning {
				SendDriverStats(s)
			}
		default:
		}
	}
}

func (s *Simulation) SetParameters(sf SettingsFormat) {

}

// func (s *Simulation) Run2() {
// 	SendTimer(s)
// }

// TODO: Reputation
func (s *Simulation) ComputeAverageValue(d *DriverAgent) float64 {
	var accumulatedTaskValue float64 = 0
	var totalDriversWithTask = 0

	for _, v := range s.DriverAgents {
		if v.Id != d.Id {
			// fmt.Printf("[ComputeAverageValue]Driver %d has Task %v with value of %v \n",
			// 	v.Id,
			// 	v.CurrentTask.Id,
			// 	v.CurrentTask.FinalValue,
			// )
			accumulatedTaskValue = v.CurrentTask.FinalValue + accumulatedTaskValue
			totalDriversWithTask++
		}
	}
	averageTaskValue := float64(accumulatedTaskValue) / float64(totalDriversWithTask)

	if math.IsNaN(averageTaskValue) {
		return 0
	}
	//fmt.Printf("[ComputeAverageValue]Final Value: %v \n", averageTaskValue)
	return averageTaskValue
}

// e.g 2,0,{environmentId},{DriverId}
// THIS METHOD IS UNUSED
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
	pickupLocation := ConvertToLatLng(command.([]interface{})[2].([]interface{}))
	dropoffLocation := ConvertToLatLng(command.([]interface{})[3].([]interface{}))
	distance := command.([]interface{})[4].(float64)

	r := RecieveFormat{
		Command:       3,
		CommandSecond: 1,
		Data: CorrectedLocation{
			StartCoordinate: pickupLocation,
			EndCoordinate:   dropoffLocation,
			Distance:        distance,
		},
	}

	s.OM.Recieve <- r
}

func SendEnvGeoJSON(s *Simulation) {
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
	}

	sendformat := &SendFormat{
		Command:       1,
		CommandSecond: 0,
		Data:          geojson,
	}

	e, err := json.Marshal(sendformat)
	if err != nil {
		fmt.Println(err)
	}
	s.Send <- string(e)
}

func SendGeoJSON(s *Simulation) {

	geojson := &GeoJSONFormat{
		Type:     "FeatureCollection",
		Features: make([]Feature, 0),
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

	sendformat := &SendFormat{
		Command:       1,
		CommandSecond: 1,
		Data:          geojson,
	}

	e, err := json.Marshal(sendformat)
	if err != nil {
		fmt.Println(err)
	}
	s.Send <- string(e)

}

func SendTaskGeoJSON(s *Simulation) {
	geojson := &GeoJSONFormat{
		Type:     "FeatureCollection",
		Features: make([]Feature, 0),
	}

	for _, v := range s.Environments {
		v.TaskMutex.Lock()
		for _, v2 := range v.Tasks {
			if (v2.WaitEnd == time.Time{}) {
				feature := Feature{
					Type: "Feature",
					Geometry: Geometry{
						Type:        "Point",
						Coordinates: latlngToArrayFloat(v2.PickUpLocation),
					},
					Properties: Properties{
						Type: "Task",
						Information: TaskFormat{
							Id:            v2.Id,
							EnvironmentId: v2.EnvironmentId,
							StartPosition: latlngToArrayFloat(v2.PickUpLocation),
							EndPosition:   latlngToArrayFloat(v2.DropOffLocation),
							WaitStart:     v2.WaitStart,
							WaitEnd:       v2.WaitEnd,
							TaskEnd:       v2.TaskEnded,
							Value:         v2.FinalValue,
							Distance:      v2.Distance,
						},
					},
				}
				geojson.Features = append(geojson.Features, feature)
			}

		}
		v.TaskMutex.Unlock()
	}

	sendformat := &SendFormat{
		Command:       1,
		CommandSecond: 2,
		Data:          geojson,
	}

	e, err := json.Marshal(sendformat)
	if err != nil {
		fmt.Println(err)
	}
	s.Send <- string(e)
}

func SendDriverStats(s *Simulation) {
	count := 0
	count2 := 0
	count3 := 0
	simTime := s.SimulationTime.Format("3:4:5PM")
	driversRegret := make([]DriverRegretFormat, 0)

	for _, v := range s.DriverAgents {
		if v.Status == Roaming || v.Status == Allocating || v.Status == Matching {
			count++
		}
		if v.Status == Fetching {
			count2++
		}
		if v.Status == Travelling {
			count3++
		}

		drf := &DriverRegretFormat{
			EnvironmentId: v.E.Id,
			DriverId:      v.Id,
			Regret:        v.Regret,
		}

		driversRegret = append(driversRegret, *drf)
	}

	statsInfo := &StatsFormat{
		Time:              simTime,
		RoamingDrivers:    count,
		FetchingDrivers:   count2,
		TravellingDrivers: count3,
	}

	// 1,3
	sendformat := &SendFormat{
		Command:       1,
		CommandSecond: 3,
		Data:          statsInfo,
	}

	e, err := json.Marshal(sendformat)
	if err != nil {
		fmt.Println(err)
	}
	s.Send <- string(e) // send StatsFormat

	regretStatsInfo := &StatsRegretFormat{
		Time:          simTime,
		DriversRegret: driversRegret,
	}

	// 1,4
	sendformat2 := &SendFormat{
		Command:       1,
		CommandSecond: 4,
		Data:          regretStatsInfo,
	}

	f, err := json.Marshal(sendformat2)
	if err != nil {
		fmt.Println(err)
	}
	s.Send <- string(f) // send StatsRegretFormat

}

func (s *Simulation) StartTimer() {
	s.isRunning = true
	for range s.Ticker {
		s.SimulationTime = s.SimulationTime.Add(1 * time.Minute)
		//fmt.Printf("[Time by StartTimer]%v\n", s.SimulationTime)
	}
}
