package dispatchsim

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/pquerna/ffjson/ffjson"
)

type Simulation struct {
	isRunning             bool
	Environments          map[int]*Environment
	DriverAgents          map[int]*DriverAgent
	DriverAgentMutex      sync.RWMutex
	OM                    *OrderManager
	RN                    *RoadNetwork
	MasterSpeed           time.Duration
	Recieve               chan string // recieve from websocket
	Send                  chan string // send to websocket
	OrderQueue            chan Order
	UpdateMap             bool          // settings
	UpdateMapSpeed        time.Duration // settings
	DispatcherSpeed       time.Duration
	UpdateStatsSpeed      time.Duration
	TickerTime            int
	Ticker                <-chan time.Time
	AddTime               time.Duration
	RegretTickerTime      time.Duration
	SimulationTime        time.Time
	TaskParameters        TaskParametersFormat
	DispatcherParameters  DispatcherParametersFormat
	VirusParameters       VirusParameters
	DriverParameters      DriverParamatersFormat
	StartDrivers          chan interface{} // start
	StartDispatchers      chan interface{} // start
	Stop                  chan interface{} // stop the simuation
	StatsVirus            []StatsVirusFormat
	StatsDriver           []StatsIndividualDrivers
	CaptureSimulationTime time.Time
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
		SimilarReputation: 5,
	}

	defaultDriverParameters := DriverParamatersFormat{
		TravellingMode: "node",
		TravelInterval: 50,
		SpeedKmPerHour: 120,
	}

	defaultVirusParams := VirusParameters{
		InitialInfectedDriversPercentage: 10,
		InfectedTaskPercentage:           10,
		EvolveProbability:                []float64{0.5, 1},
		SpreadProbability:                []float64{4, 9, 13},
		DriverMask:                       10,
		PassengerMask:                    10,
		MaskEffectiveness:                95,
	}

	tickerTime := 10 // make it adjustable
	tickerTime2 := time.Duration(tickerTime)
	AddTime := time.Duration(5000)                    // half a second of simulation time
	RejectTickerTime := time.Duration(tickerTime * 2) // 1sec of simulation time

	return Simulation{
		isRunning:             false,
		Environments:          make(map[int]*Environment),
		DriverAgents:          make(map[int]*DriverAgent),
		DriverAgentMutex:      sync.RWMutex{},
		RN:                    SetupRoadNetwork2(),
		MasterSpeed:           50,
		Recieve:               make(chan string, 10000),
		Send:                  make(chan string, 10000),
		OrderQueue:            make(chan Order, 1000),
		UpdateMap:             true, // set true to update mapbox
		UpdateMapSpeed:        200,  // update speed to mapbox
		DispatcherSpeed:       5000,
		UpdateStatsSpeed:      1000,
		TickerTime:            tickerTime,                                // 100ms
		Ticker:                time.Tick(tickerTime2 * time.Millisecond), // 100ms real time-> half a second in simulation time
		AddTime:               AddTime,
		RegretTickerTime:      RejectTickerTime,
		SimulationTime:        time.Date(2020, 1, 23, 00, 05, 0, 0, time.Local), // TODO: Make adjustable
		TaskParameters:        defaultTaskParameters,
		DispatcherParameters:  defaultDispatcherParameters,
		VirusParameters:       defaultVirusParams,
		DriverParameters:      defaultDriverParameters,
		StartDrivers:          make(chan interface{}),
		StartDispatchers:      make(chan interface{}),
		Stop:                  make(chan interface{}),
		StatsVirus:            make([]StatsVirusFormat, 0),
		StatsDriver:           make([]StatsIndividualDrivers, 0),
		CaptureSimulationTime: time.Date(2020, 1, 23, 00, 05, 0, 0, time.Local),
	}
}

func (s *Simulation) Run() {
	var environmentId = 1 // starting id
	var noOfDrivers = 0
	var startingDriverCount = 1 // starting id of driver

	var start = false
	var startOrderRetriever = false

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
					s.VirusParameters = sf.VirusParameters
					s.DriverParameters = sf.DriverParameters
					fmt.Printf("Task: %v\n", sf.TaskParameters)
					fmt.Printf("Dispatcher: %v\n", sf.DispatcherParameters)
					fmt.Printf("Driver: %v\n", sf.DriverParameters)
					fmt.Printf("Virus: %v\n", sf.VirusParameters)
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
				CreateMultipleVirusDrivers(startingDriverCount, inputNoOfDrivers, &env, s)
				go env.Run()                                                 // run environment
				startingDriverCount = startingDriverCount + inputNoOfDrivers // update driver id count
				environmentId++
				s.SendMessageToClient("Generating " + strconv.Itoa(inputNoOfDrivers) + " drivers")
				SendEnvGeoJSON(s) // send polygon to client // TODO: settle this case in client
			case 2: // drivers
			case 3: // order distributor
				commandTypeLevelTwo := command.([]interface{})[1].(float64)
				switch commandTypeLevelTwo {
				case 0:
					if len(s.Environments) > 0 && start == false {
						close(s.StartDrivers)     // start drivers
						close(s.StartDispatchers) // start dispatcher
						go s.StartTimer()
						start = true
						s.SendMessageToClient("Simulation started")
						initLogger()
					} else {
						if start == true {
							//s.SendMessageToClient("Invalid command as simulation has started")
							close(s.Stop)
							s.isRunning = false
						} else {
							s.SendMessageToClient("Please create an environment with drivers")
						}

					}
				case 1: // Generating virus csv
					if s.isRunning == false && start == true {
						s.SendMessageToClient("Generating virus csv")
						s.GenerateVirusCSV()
					} else {
						s.SendMessageToClient("Unable to generate virus csv. Simulation running")
					}
				case 2:
					// initializing order retriever
					if startOrderRetriever == false {
						om := SetupOrderRetrieve(s)
						s.OM = &om
						go om.RunOrderRetriever()
						s.SendMessageToClient("Retrieving orders")
					}
				case 3:
					if s.isRunning == false && start == true {
						s.SendMessageToClient("Generating driver's stats csv")
						s.GenerateIndividualDriverCSV()
					} else {
						s.SendMessageToClient("Unable to generate virus csv. Simulation running")
					}
				}
			}
		}
	}
	fmt.Println("[Simulation]Ended")
}

func (s *Simulation) SendMapData() {
	fmt.Printf("[Simulator]SendMapData started \n")
	tick := time.Tick((s.UpdateMapSpeed) * time.Millisecond)
	for {
		select {
		case <-s.Stop:
			return
		case <-tick:
			if s.UpdateMap && s.isRunning { // send updates when there is Driver Agent available and environment placed
				SendVirusDriversGeoJSON(s)
				SendVirusTasksJSON(s)
			}
		}
	}
}

func (s *Simulation) SendStats() {
	fmt.Printf("[Simulator]sendStats started \n")
	tick := time.Tick((s.UpdateStatsSpeed) * time.Millisecond)
	for {
		select {
		case <-s.Stop:
			return
		case <-tick:
			if s.isRunning {
				SendDriverStats(s)
				SendEnvironmentStats(s)

				if s.CaptureSimulationTime.Hour() != s.SimulationTime.Hour() || s.CaptureSimulationTime.Minute() != s.SimulationTime.Minute() {
					s.CaptureSimulationTime = s.SimulationTime
				}
			}
		}
	}
}

func (s *Simulation) ComputeAverageValue(d *DriverAgent) float64 {
	var accumulatedTaskValue float64 = 0
	var totalDriversWithTask = 0

	upperLimit := d.Reputation + s.DispatcherParameters.SimilarReputation
	lowerLimit := d.Reputation - s.DispatcherParameters.SimilarReputation
	for _, v := range s.DriverAgents {
		if v.Id != d.Id && lowerLimit <= v.Reputation && v.Reputation <= upperLimit {
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

// unused
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

/// unused
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

	//start := time.Now()

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
	//elapsed := time.Since(start)
	//log.Printf("Sending drivers' geojson %s", elapsed)

}

func SendVirusTasksJSON(s *Simulation) {
	geojson := &GeoJSONFormat{
		Type:     "FeatureCollection",
		Features: make([]Feature, 0),
	}

	for _, v := range s.Environments {
		//fmt.Printf("TasksToDrivers %v\n", v.TasksToDrivers)
		//fmt.Printf("DriversToTasks %v\n", v.DriversToTasks)
		v.TaskMutex.Lock()
		for _, v2 := range v.Tasks {
			if (v2.WaitEnd == time.Time{} && v2.Valid == true && v2.Appear == true) {
				feature := Feature{
					Type: "Feature",
					Geometry: Geometry{
						Type:        "Point",
						Coordinates: latlngToArrayFloat(v2.PickUpLocation),
					},
					Properties: Properties{
						Information: TaskVirusFormat{
							Id:            v2.Id,
							Type:          "Task",
							Virus:         v2.Virus,
							StartPosition: latlngToArrayFloat(v2.PickUpLocation),
							EndPosition:   latlngToArrayFloat(v2.DropOffLocation),
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
		CommandSecond: 3,
		Data:          geojson,
	}

	e, err := ffjson.Marshal(sendformat)
	if err != nil {
		fmt.Println(err)
	}

	s.Send <- string(e)

}

func SendVirusDriversGeoJSON(s *Simulation) {
	geojson := &GeoJSONFormat{
		Type:     "FeatureCollection",
		Features: make([]Feature, 0),
	}

	//start := time.Now()

	s.DriverAgentMutex.Lock()
	for _, v := range s.DriverAgents {
		feature := Feature{
			Type: "Feature",
			Geometry: Geometry{
				Type:        "Point",
				Coordinates: latlngToArrayFloat(v.CurrentLocation),
			},
			Properties: Properties{
				Information: DriverVirusFormat{
					Id:          v.Id,
					Type:        "Driver",
					Virus:       v.Virus,
					Status:      v.Status,
					CurrentTask: v.CurrentTask.Id,
				},
			},
		}
		geojson.Features = append(geojson.Features, feature)
	}
	s.DriverAgentMutex.Unlock()

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
	//elapsed := time.Since(start)
	//log.Printf("Sending drivers' geojson %s", elapsed)

}

// For charts (4,5)
// roam,pick,travelling, and regrets of drivers
func SendDriverStats(s *Simulation) {
	count := 0
	count2 := 0
	count3 := 0
	simTime := s.SimulationTime.Format("3:4:5PM")
	driversRegret := make([]DriverRegretFormat, 0)

	first := false
	var minEarning float64 = 0
	var totalEarning float64 = 0
	var maxEarning float64 = 0
	validDrivers := 0

	firstwithTasks := false
	var minCurrentEarning float64 = 0
	var totalCurrentEarning float64 = 0
	var maxCurrentEarning float64 = 0
	validCurrentDrivers := 0

	// for CSV - driver stats
	sid := &StatsIndividualDrivers{
		Time:        simTime,
		DriverStats: make([]DriverRegretFormat, 0),
	}

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

		// min, average, max
		if v.Valid {
			validDrivers++

			// Min, average, max of total earnings from valid drivers
			if !first {
				minEarning = v.TotalEarnings
				maxEarning = v.TotalEarnings
				//fmt.Printf("[SendData]%v %v\n", minEarning, maxEarning)
				first = true
			}

			if v.TotalEarnings < minEarning {
				minEarning = v.TotalEarnings
			}

			if v.TotalEarnings > maxEarning {
				maxEarning = v.TotalEarnings
			}

			totalEarning += v.TotalEarnings

			// Min, average, max of valid drivers who have a tasks on hand
			if v.CurrentTask.Id != "null" && !firstwithTasks {
				minCurrentEarning = v.CurrentTask.FinalValue
				maxCurrentEarning = v.CurrentTask.FinalValue
				firstwithTasks = true
			}

			if v.CurrentTask.Id != "null" {
				if v.CurrentTask.FinalValue < minCurrentEarning {
					minCurrentEarning = v.CurrentTask.FinalValue
				}
				if v.CurrentTask.FinalValue > maxCurrentEarning {
					//fmt.Printf("[SendData] New Max - Task %v\n", v.CurrentTask.Id)
					maxCurrentEarning = v.CurrentTask.FinalValue
				}
				totalCurrentEarning += v.CurrentTask.FinalValue
				validCurrentDrivers++
			}
		}

		drf := &DriverRegretFormat{
			EnvironmentId:    v.E.Id,
			DriverId:         v.Id,
			Regret:           v.Regret,
			Motivation:       v.Motivation,
			Reputation:       v.Reputation,
			Fatigue:          v.Fatigue,
			RankingIndex:     v.GetRawRankingIndex(),
			CurrentTaskValue: v.CurrentTask.FinalValue,
			TotalEarnings:    v.TotalEarnings,
		}
		sid.DriverStats = append(sid.DriverStats, *drf)
		driversRegret = append(driversRegret, *drf)
	}

	// csv
	if s.CaptureSimulationTime.Hour() != s.SimulationTime.Hour() || s.CaptureSimulationTime.Minute() != s.SimulationTime.Minute() {

		sort.SliceStable(sid.DriverStats, func(i, j int) bool {
			return sid.DriverStats[i].DriverId < sid.DriverStats[j].DriverId
		})

		s.StatsDriver = append(s.StatsDriver, *sid)
	}

	var averageTotalCurrentEarning float64 = 0
	if totalCurrentEarning == 0 {
		maxCurrentEarning = 0
		averageTotalCurrentEarning = 0
		minCurrentEarning = 0
	} else {
		averageTotalCurrentEarning = totalCurrentEarning / float64(validCurrentDrivers)
	}

	//fmt.Printf("[SendData]Updated %v %v\n", minEarning, maxEarning)

	statsInfo := &StatsDriverStatusFormat{
		Time:              simTime,
		RoamingDrivers:    count,
		FetchingDrivers:   count2,
		TravellingDrivers: count3,
	}

	sef := &StatsEarningFormat{
		Time:    simTime,
		Max:     maxEarning,
		Average: totalEarning / float64(validDrivers),
		Min:     minEarning,
	}

	scef := &StatsCurrentEarningFormat{
		Time:    simTime,
		Max:     maxCurrentEarning,
		Average: averageTotalCurrentEarning,
		Min:     minCurrentEarning,
	}

	// 1,4
	sendformat := &SendFormat{
		Command:       1,
		CommandSecond: 4,
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

	// 1,5
	sendformat2 := &SendFormat{
		Command:       1,
		CommandSecond: 5,
		Data:          regretStatsInfo,
	}

	f, err := json.Marshal(sendformat2)
	if err != nil {
		fmt.Println(err)
	}
	s.Send <- string(f) // send StatsRegretFormat - change to send DriverStats

	sendformat3 := &SendFormat{
		Command:       1,
		CommandSecond: 7,
		Data:          sef,
	}

	g, err := json.Marshal(sendformat3)
	if err != nil {
		fmt.Println(err)
	}
	s.Send <- string(g) // send StatsEarningFormat

	sendformat4 := &SendFormat{
		Command:       1,
		CommandSecond: 8,
		Data:          scef,
	}

	h, err := json.Marshal(sendformat4)
	if err != nil {
		fmt.Println(err)
	}
	s.Send <- string(h) // send StatsEarningFormat

}

func SendEnvironmentStats(s *Simulation) {
	simTime := s.SimulationTime.Format("3:4:5PM")
	tasksToDrivers := 0
	driversToTasks := 0
	for _, e := range s.Environments {
		tasksToDrivers = tasksToDrivers + e.TasksToDrivers
		driversToTasks = driversToTasks + e.DriversToTasks
	}

	svf := &StatsVirusFormat{
		Time:           simTime,
		TasksToDrivers: tasksToDrivers,
		DriversToTasks: driversToTasks,
	}

	if s.CaptureSimulationTime.Hour() != s.SimulationTime.Hour() || s.CaptureSimulationTime.Minute() != s.SimulationTime.Minute() {
		s.StatsVirus = append(s.StatsVirus, *svf)
	}

	sendformat := &SendFormat{
		Command:       1,
		CommandSecond: 6,
		Data:          svf,
	}

	f, err := json.Marshal(sendformat)
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

func (s *Simulation) GenerateVirusCSV() {
	result := fmt.Sprintf("./src/github.com/harrizontal/dispatchserver/assets/virus/virus_data_%v_%v_%v_%v_%v.csv",
		s.VirusParameters.InitialInfectedDriversPercentage,
		s.VirusParameters.InfectedTaskPercentage,
		s.VirusParameters.DriverMask,
		s.VirusParameters.PassengerMask,
		s.VirusParameters.MaskEffectiveness)

	csvFile, err := os.Create(result)

	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	csvwriter := csv.NewWriter(csvFile)
	count := 0
	for _, row := range s.StatsVirus {
		data := []string{strconv.Itoa(count),
			row.Time,
			strconv.Itoa(row.DriversToTasks),
			strconv.Itoa(row.TasksToDrivers)}

		csvwriter.Write(data)
		count++
	}

	csvwriter.Flush()
	csvFile.Close()

	s.SendMessageToClient("Virus data generated")
}

func (s *Simulation) GenerateIndividualDriverCSV() {
	path := "./src/github.com/harrizontal/dispatchserver/assets/driver/"

	result := fmt.Sprintf(path + "driver_data_Regret.csv")
	fatiguePath := fmt.Sprintf(path + "driver_data_Fatigue.csv")
	currentTaskValuePath := fmt.Sprintf(path + "driver_data_CurrentTaskValue.csv")
	rankingIndexPath := fmt.Sprintf(path + "driver_data_RankingIndex.csv")
	currentTotalEarningsPath := fmt.Sprintf(path + "driver_data_TotalEarnings.csv")
	reputationPath := fmt.Sprintf(path + "driver_data_Reputation.csv")

	csvFile, err := os.Create(result)
	csvFatigue, _ := os.Create(fatiguePath)
	csvCurrentTaskValue, _ := os.Create(currentTaskValuePath)
	csvRankingIndex, _ := os.Create(rankingIndexPath)
	csvTotalEarnings, _ := os.Create(currentTotalEarningsPath)
	csvReputation, _ := os.Create(reputationPath)

	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	csvwriter := csv.NewWriter(csvFile)
	csvFatigueWriter := csv.NewWriter(csvFatigue)
	csvCurrentTaskValueWriter := csv.NewWriter(csvCurrentTaskValue)
	csvRankingIndexWriter := csv.NewWriter(csvRankingIndex)
	csvTotalEarningsWriter := csv.NewWriter(csvTotalEarnings)
	csvReputationWriter := csv.NewWriter(csvReputation)

	count := 0

	// store valid drivers
	validDrivers := make(map[int]struct{})
	for _, d := range s.DriverAgents {
		if d.Valid {
			validDrivers[d.Id] = struct{}{}
		} else {
			fmt.Printf("Driver %d is invalid. Will be remove from csv\n", d.Id)
		}
	}

	// write header
	data := []string{"count", "time"}
	for _, d := range s.StatsDriver[0].DriverStats {
		_, ok := validDrivers[d.DriverId]
		if ok {
			data = append(data, strconv.Itoa(d.DriverId))
		}
	}
	csvwriter.Write(data)
	csvFatigueWriter.Write(data)
	csvCurrentTaskValueWriter.Write(data)
	csvRankingIndexWriter.Write(data)
	csvTotalEarningsWriter.Write(data)
	csvReputationWriter.Write(data)
	// end of writing header

	for _, row := range s.StatsDriver {
		data := []string{strconv.Itoa(count),
			row.Time}

		dataFatigue := data
		dataCTV := data
		dataRankingIndex := data
		dataTotalEarnings := data
		dataReputation := data

		for _, d := range row.DriverStats {
			_, ok := validDrivers[d.DriverId]
			if ok {
				//s := fmt.Sprintf("%f", d.Fatigue)
				data = append(data, strconv.Itoa(int(d.Regret)))
				dataFatigue = append(dataFatigue, strconv.Itoa(int(d.Fatigue)))
				dataCTV = append(dataCTV, fmt.Sprintf("%f", d.CurrentTaskValue))
				dataRankingIndex = append(dataRankingIndex, strconv.Itoa(int(d.RankingIndex)))
				dataTotalEarnings = append(dataTotalEarnings, fmt.Sprintf("%f", d.TotalEarnings))
				dataReputation = append(dataReputation, fmt.Sprintf("%f", d.Reputation))
			}
		}
		csvwriter.Write(data)
		csvFatigueWriter.Write(dataFatigue)
		csvCurrentTaskValueWriter.Write(dataCTV)
		csvRankingIndexWriter.Write(dataRankingIndex)
		csvTotalEarningsWriter.Write(dataTotalEarnings)
		csvReputationWriter.Write(dataReputation)

		count++
	}

	csvwriter.Flush()
	csvFatigueWriter.Flush()
	csvCurrentTaskValueWriter.Flush()
	csvRankingIndexWriter.Flush()
	csvTotalEarningsWriter.Flush()
	csvReputationWriter.Flush()

	s.SendMessageToClient("Finished generating driver's stats")
}
