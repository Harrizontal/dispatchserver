package dispatchsim

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/pquerna/ffjson/ffjson"
)

type Simulation struct {
	isRunning            bool
	Environments         map[int]*Environment
	DriverAgents         map[int]*DriverAgent
	DriverAgentMutex     sync.RWMutex
	OM                   *OrderManager
	RN                   *RoadNetwork
	MasterSpeed          time.Duration
	Recieve              chan string // recieve from websocket
	Send                 chan string // send to websocket
	OrderQueue           chan Order
	UpdateMap            bool          // settings
	UpdateMapSpeed       time.Duration // settings
	DispatcherSpeed      time.Duration
	UpdateStatsSpeed     time.Duration
	TickerTime           time.Duration
	Ticker               <-chan time.Time
	SimulationTime       time.Time
	TaskParameters       TaskParametersFormat
	DispatcherParameters DispatcherParametersFormat
	StartDrivers         chan interface{}
	StartDispatchers     chan interface{}
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

	tickerTime := time.Duration(100) // make it adjustable

	return Simulation{
		isRunning:            false,
		Environments:         make(map[int]*Environment),
		DriverAgents:         make(map[int]*DriverAgent),
		DriverAgentMutex:     sync.RWMutex{},
		RN:                   SetupRoadNetwork2(),
		MasterSpeed:          50,
		Recieve:              make(chan string, 10000),
		Send:                 make(chan string, 10000),
		OrderQueue:           make(chan Order, 1000),
		UpdateMap:            true, // set true to update mapbox
		UpdateMapSpeed:       1000, // update speed to mapbox
		DispatcherSpeed:      5000,
		UpdateStatsSpeed:     1000,
		TickerTime:           tickerTime,
		Ticker:               time.Tick(tickerTime * time.Millisecond),         // TODO: Make adjustable - 20 millisecond -> increase by 1 min
		SimulationTime:       time.Date(2020, 1, 23, 00, 05, 0, 0, time.Local), // TODO: Make adjustable
		TaskParameters:       defaultTaskParameters,
		DispatcherParameters: defaultDispatcherParameters,
		StartDrivers:         make(chan interface{}),
		StartDispatchers:     make(chan interface{}),
	}
}

func (s *Simulation) Run() {
	var environmentId = 1 // starting id
	var noOfDrivers = 0
	var startingDriverCount = 1 // starting id of driver

	var k = false

	go s.SendMapData()
	go s.SendStats()

	for {
		select {
		case recieveCommand := <-s.Recieve:
			command := stringToArrayString(recieveCommand)

			commandType := command.([]interface{})[0].(float64)
			switch commandType {
			case 0: // pause
				commandTypeLevelTwo := command.([]interface{})[1].(float64)
				switch commandTypeLevelTwo {
				case 0: // pause
				case 1: // settings
					settings := command.([]interface{})[2]
					byteData, _ := json.Marshal(settings)
					var sf SettingsFormat
					if err := json.Unmarshal(byteData, &sf); err != nil {
						log.Fatal(err)
					}
					s.TaskParameters = sf.TaskParameters
					s.DispatcherParameters = sf.DispatcherParameters
					s.SendMessageToClient("Parameters applied")
				}

				fmt.Printf("[Simulator]Parameters applied\n")
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
				SendEnvGeoJSON(s) // send polygon to client // TODO: settle this case in client
			case 2: // drivers
			case 3: // order distributor
				commandTypeLevelTwo := command.([]interface{})[1].(float64)
				switch commandTypeLevelTwo {
				case 0:
					if len(s.Environments) > 0 && k == false {
						k = true
						close(s.StartDrivers)     // start drivers
						close(s.StartDispatchers) // start dispatcher
						go s.StartTimer()
						//go s.OM.runOrderDistributer()
					} else {
						s.Send <- "[Simulator]Cannot intailize order distributor"
					}
				case 1: // pickup lnglat and drop off lnglat in terms of waypoint
					//fmt.Printf("accessing 3,1\n")
					//sendCorrectedLocation(command, s)
				case 2:
					// initializing order retriever
					om := SetupOrderRetrieve(s)
					s.OM = &om
					go om.runOrderRetrieve2()

				}
			}
			//fmt.Println("[Simulation]Running")
		}
		//fmt.Println("[Simulation]Running2")
	}
	fmt.Println("[Simulation]Ended")
}

func (s *Simulation) SendMapData() {
	fmt.Printf("[Simulator]SendMapData started \n")
	tick := time.Tick((s.UpdateMapSpeed) * time.Millisecond)
	for {
		select {
		case <-tick:
			//fmt.Printf("[Simulator]Sending map data\n")
			if s.UpdateMap && s.isRunning { // send updates when there is Driver Agent available and environment placed
				go SendDriversGeoJSON(s)
				go SendTasksJSON(s)
			}
			//default:

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
				SendTasksJSON(s)
			}
			//default:
		}
	}
}

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

func (s *Simulation) GetMinMaxReputationFatigue() [2][2]float64 {
	var first bool = false
	var minReputation float64 = 0
	var maxReputation float64 = 0
	var minFatigue float64 = 0
	var maxFatigue float64 = 0
	s.DriverAgentMutex.Lock()
	for _, v := range s.DriverAgents {
		if !first {
			minReputation = v.Reputation
			maxReputation = v.Reputation
			minFatigue = v.Fatigue
			maxFatigue = v.Fatigue
			first = true
		}

		// Reputation
		if v.Reputation < minReputation {
			minReputation = v.Reputation
		}

		if v.Reputation > maxReputation {
			maxReputation = v.Reputation
		}

		// Fatigue
		if v.Fatigue < minFatigue {
			minFatigue = v.Fatigue
		}

		if v.Fatigue > maxFatigue {
			maxFatigue = v.Fatigue
		}
	}
	s.DriverAgentMutex.Unlock()
	return [2][2]float64{{minReputation, maxReputation}, {minFatigue, maxFatigue}}
}

// // e.g 2,0,{environmentId},{DriverId}
// // THIS METHOD IS UNUSED
// func sendRandomPointDriver(command interface{}, s *Simulation, eId int, dId int) {
// 	latLng := ConvertToLatLng(command.([]interface{})[4].([]interface{}))
// 	s.Environments[eId].DriverAgents[dId].Recieve <- Message{CommandType: 2, CommandSecondType: 0, LatLng: latLng}
// }

// func sendIntializationDriver(command interface{}, s *Simulation, eId int, dId int) {
// 	// fmt.Printf("%T %[1]v \n", command.([]interface{})[6].([]interface{}))
// 	startLocation := ConvertToLatLng(command.([]interface{})[4].([]interface{}))
// 	destLocation := ConvertToLatLng(command.([]interface{})[5].([]interface{}))
// 	waypoints := ConvertToArrayLatLng(command.([]interface{})[6].([]interface{}))

// 	message := Message{
// 		CommandType:       2,
// 		CommandSecondType: 1,
// 		StartDestinationWaypoint: StartDestinationWaypoint{
// 			StartLocation:       startLocation,
// 			DestinationLocation: destLocation,
// 			Waypoint:            waypoints,
// 		},
// 	}

// 	s.DriverAgents[dId].Recieve <- message
// }

// func sendWaypointsDriver(command interface{}, s *Simulation, eId int, dId int) {

// 	message := Message{
// 		CommandType:       2,
// 		CommandSecondType: 2,
// 		Waypoint:          ConvertToArrayLatLng(command.([]interface{})[4].([]interface{})),
// 	}

// 	s.DriverAgents[dId].Recieve <- message
// }

// func sendGenerateResultDriver(command interface{}, s *Simulation, eId int, dId int) {
// 	success := command.([]interface{})[4].(bool)
// 	//fmt.Printf("%T %[1]v\n", success)
// 	message := Message{
// 		CommandType:       2,
// 		CommandSecondType: 3,
// 		Success:           success,
// 	}

// 	s.DriverAgents[dId].Recieve <- message
// }

// func sendMoveResultDriver(command interface{}, s *Simulation, eId int, dId int) {
// 	location := ConvertToLatLng(command.([]interface{})[4].([]interface{}))
// 	//fmt.Printf("%T %[1]v\n", success)
// 	message := Message{
// 		CommandType:       2,
// 		CommandSecondType: 4,
// 		LocationArrived:   location,
// 		Success:           true, //TODO!!! remove this!
// 	}

// 	s.DriverAgents[dId].Recieve <- message
// }

// func sendRandomDestinationWaypoint(command interface{}, s *Simulation, eId int, dId int) {
// 	// fmt.Printf("%T %[1]v \n", command.([]interface{})[6].([]interface{}))
// 	startLocation := ConvertToLatLng(command.([]interface{})[4].([]interface{}))
// 	destLocation := ConvertToLatLng(command.([]interface{})[5].([]interface{}))
// 	waypoints := ConvertToArrayLatLng(command.([]interface{})[6].([]interface{}))

// 	message := Message{
// 		CommandType:       2,
// 		CommandSecondType: 5,
// 		StartDestinationWaypoint: StartDestinationWaypoint{
// 			StartLocation:       startLocation,
// 			DestinationLocation: destLocation,
// 			Waypoint:            waypoints,
// 		},
// 	}

// 	s.DriverAgents[dId].Recieve <- message
// }

// func sendCorrectedLocation(command interface{}, s *Simulation) {
// 	pickupLocation := ConvertToLatLng(command.([]interface{})[2].([]interface{}))
// 	dropoffLocation := ConvertToLatLng(command.([]interface{})[3].([]interface{}))
// 	distance := command.([]interface{})[4].(float64)

// 	r := RecieveFormat{
// 		Command:       3,
// 		CommandSecond: 1,
// 		Data: CorrectedLocation{
// 			StartCoordinate: pickupLocation,
// 			EndCoordinate:   dropoffLocation,
// 			Distance:        distance,
// 		},
// 	}

// 	s.OM.Recieve <- r
// }

func SendEnvGeoJSON(s *Simulation) {
	geojson := &GeoJSONFormat{
		Type:     "FeatureCollection",
		Features: make([]Feature, 0),
	}

	for _, v := range s.Environments {
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
		CommandSecond: 2,
		Data:          geojson,
	}

	e, err := json.Marshal(sendformat)
	if err != nil {
		fmt.Println(err)
	}

	s.Send <- string(e)

}

func SendTasksJSON(s *Simulation) {
	geojson := &GeoJSONFormat{
		Type:     "FeatureCollection",
		Features: make([]Feature, 0),
	}

	for _, v := range s.Environments {
		//v.TaskMutex.Lock()
		for _, v2 := range v.Tasks {
			if (v2.WaitEnd == time.Time{} && v2.Valid == true && v2.Appear == true) {
				feature := Feature{
					Type: "Feature",
					Geometry: Geometry{
						Type:        "Point",
						Coordinates: latlngToArrayFloat(v2.PickUpLocation),
					},
					Properties: Properties{
						Information: TaskFormat{
							Id:            v2.Id,
							EnvironmentId: v2.EnvironmentId,
							Type:          "Task",
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
		//v.TaskMutex.Unlock()
	}

	sendformat := &SendFormat{
		Command:       1,
		CommandSecond: 5,
		Data:          geojson,
	}

	e, err := ffjson.Marshal(sendformat)
	if err != nil {
		fmt.Println(err)
	}

	s.Send <- string(e)

}

func SendDriversGeoJSON(s *Simulation) {
	geojson := &GeoJSONFormat{
		Type:     "FeatureCollection",
		Features: make([]Feature, 0),
	}

	start := time.Now()

	for _, v := range s.DriverAgents {
		feature := Feature{
			Type: "Feature",
			Geometry: Geometry{
				Type:        "Point",
				Coordinates: latlngToArrayFloat(v.CurrentLocation),
			},
			Properties: Properties{
				Information: DriverFormat{
					Id:            v.Id,
					EnvironmentId: v.E.Id,
					Type:          "Driver",
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

	e, err := ffjson.Marshal(sendformat)
	if err != nil {
		fmt.Println(err)
	}

	s.Send <- string(e)
	elapsed := time.Since(start)
	log.Printf("Sending drivers' geojson %s", elapsed)

}

// for charts
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

func (s *Simulation) SendMessageToClient(message string) {

	sendformat := &SendFormat{
		Command:       1,
		CommandSecond: 0,
		Data:          message,
	}

	f, err := json.Marshal(sendformat)
	if err != nil {
		fmt.Println(err)
	}
	s.Send <- string(f)

}
func (s *Simulation) StartTimer() {
	s.isRunning = true
	for range s.Ticker {
		s.SimulationTime = s.SimulationTime.Add(5000 * time.Millisecond) // add half a second
		//fmt.Printf("[Time by StartTimer]%v\n", s.SimulationTime)
	}
}
