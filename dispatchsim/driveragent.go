package dispatchsim

import (
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/paulmach/orb"
)

type DriverStatus int

const (
	Roaming DriverStatus = iota
	Matching
	Fetching
	Travelling
	Allocating
)

func (d DriverStatus) String() string {
	return [...]string{"Roaming", "Matching", "Fetching", "Travelling", "Allocating"}[d]
}

type DriverAgent struct {
	S                   *Simulation
	E                   *Environment
	Id                  int
	Name                string
	CurrentLocation     LatLng
	Waypoint            []LatLng
	NextLocation        LatLng
	DestinationLocation LatLng
	PendingTask         Task // store task during matching
	CurrentTask         Task // store task when fetching/travelling
	Status              DriverStatus
	TotalEarnings       float64
	Request             chan Message
	Recieve             chan Message // receive
	Recieve2            chan int
	RecieveNewRoute     chan []LatLng
	RequestNewRoute     bool
	Send                chan string // send
	TasksCompleted      int
	TaskHistory         []Task
	Motivation          float64
	Reputation          float64
	Fatigue             float64
	Regret              float64
	RankingIndex        float64
	ChangeDestination   chan string
	StartDriving        time.Time
	EndDriving          time.Time
	Valid               bool
	Virus               Virus // virus
	Mask                bool  // virus
	DriverMutex         sync.RWMutex
}

func CreateMultipleDrivers(startingDriverId, n int, e *Environment, s *Simulation) {
	for i := 0; i < n; i++ {
		driver := CreateDriver(startingDriverId, e, s)
		e.DriverAgents[startingDriverId] = &driver
		s.DriverAgents[startingDriverId] = &driver
		go e.DriverAgents[startingDriverId].ProcessTask3()
		startingDriverId++
	}
	fmt.Printf("[Environment %d]%v drivers spawned\n", e.Id, n)
}

func CreateDriver(id int, e *Environment, s *Simulation) DriverAgent {
	return DriverAgent{
		S:                 s,
		E:                 e,
		Id:                id,
		Name:              "John",
		Status:            Roaming,
		TotalEarnings:     0,
		Request:           make(chan Message),
		Recieve:           make(chan Message),
		Recieve2:          make(chan int),
		RecieveNewRoute:   make(chan []LatLng),
		RequestNewRoute:   false,
		Send:              make(chan string),
		TasksCompleted:    0,
		TaskHistory:       make([]Task, 0),
		Motivation:        100,
		Reputation:        5,
		Fatigue:           0,
		Regret:            0,
		ChangeDestination: make(chan string),
		Valid:             true,
		DriverMutex:       sync.RWMutex{},
	}
}

// virus
func CreateMultipleVirusDrivers(startingDriverId, n int, e *Environment, s *Simulation) {

	fmt.Printf("[Environment %d]Creating a total of %v drivers\n", e.Id, n)
	infected := int(math.Round(float64(s.VirusParameters.InitialInfectedDriversPercentage) / 100.00 * float64(n)))
	fmt.Printf("[Environment %d]Expected to create a total of %v infected drivers out of %v drivers\n", e.Id, infected, n)

	infectedDriversWithMask := int(math.Round(float64(s.VirusParameters.DriverMask) / 100.00 * float64(infected)))
	fmt.Printf("[Environment %d]Expected to create %v infected drivers with Mask %v\n", e.Id, infectedDriversWithMask, float64(s.VirusParameters.DriverMask))
	// spawn mild drivers w/ mask
	for i := 0; i < infectedDriversWithMask; i++ {
		driver := CreateVirusDriver(startingDriverId, e, s, Mild, true)
		e.DriverAgents[startingDriverId] = &driver
		s.DriverAgents[startingDriverId] = &driver
		fmt.Printf("1Spawning Driver %d, Virus: %v, Mask: %v\n", driver.Id, driver.Virus, driver.Mask)
		go e.DriverAgents[startingDriverId].ProcessTask3()
		startingDriverId++
	}

	// spawn mild drivers w/o mask
	for i := 0; i < (infected - infectedDriversWithMask); i++ {
		driver := CreateVirusDriver(startingDriverId, e, s, Mild, false)
		e.DriverAgents[startingDriverId] = &driver
		s.DriverAgents[startingDriverId] = &driver
		fmt.Printf("2Spawning Driver %d, Virus: %v, Mask: %v\n", driver.Id, driver.Virus, driver.Mask)
		go e.DriverAgents[startingDriverId].ProcessTask3()
		startingDriverId++
	}

	driversWithMask := int(math.Round(float64(s.VirusParameters.DriverMask) / 100.00 * float64(n-infected)))
	fmt.Printf("[Environment %d]Expected to create %v healthy drivers with Mask\n", e.Id, driversWithMask)
	// spawn normal drivers with mask
	for i := 0; i < driversWithMask; i++ {
		driver := CreateVirusDriver(startingDriverId, e, s, None, true)
		e.DriverAgents[startingDriverId] = &driver
		s.DriverAgents[startingDriverId] = &driver
		fmt.Printf("3Spawning Driver %d, Virus: %v, Mask: %v\n", driver.Id, driver.Virus, driver.Mask)
		go e.DriverAgents[startingDriverId].ProcessTask3()
		startingDriverId++
	}

	// spawn normal drivers w/o mask
	for i := 0; i < (n - infected - driversWithMask); i++ {
		driver := CreateVirusDriver(startingDriverId, e, s, None, false)
		e.DriverAgents[startingDriverId] = &driver
		s.DriverAgents[startingDriverId] = &driver
		fmt.Printf("4Spawning Driver %d, Virus: %v, Mask: %v\n", driver.Id, driver.Virus, driver.Mask)
		go e.DriverAgents[startingDriverId].ProcessTask3()
		startingDriverId++
	}
	fmt.Printf("[Environment %d]%v drivers spawned\n", e.Id, n)
}

// virus
func CreateVirusDriver(id int, e *Environment, s *Simulation, v Virus, mask bool) DriverAgent {
	return DriverAgent{
		S:                 s,
		E:                 e,
		Id:                id,
		Name:              "John",
		Status:            Roaming,
		Request:           make(chan Message),
		Recieve:           make(chan Message),
		Recieve2:          make(chan int),
		RecieveNewRoute:   make(chan []LatLng),
		RequestNewRoute:   false,
		Send:              make(chan string),
		TasksCompleted:    0,
		TaskHistory:       make([]Task, 0),
		Motivation:        100,
		Reputation:        0,
		Fatigue:           0,
		Regret:            0,
		RankingIndex:      0,
		ChangeDestination: make(chan string),
		Valid:             true,
		Virus:             v,
		Mask:              mask,
		DriverMutex:       sync.RWMutex{},
	}
}

func (d *DriverAgent) Migrate() bool {
	// if the driver's location is in the not same
	cl := d.CurrentLocation
	point := orb.Point{cl.Lng, cl.Lat} //TODO pls fix the position of LatLng to LngLat
	inside := isPointInside(d.E.Polygon, point)
	if !inside {
		for _, v := range d.E.S.Environments {
			inside2 := isPointInside(v.Polygon, point)
			if inside2 && d.Status != Allocating { // If is allocating, it should not migrate.
				d.E.DriverAgentMutex.Lock()
				delete(d.E.DriverAgents, d.Id)
				d.E.DriverAgentMutex.Unlock()

				v.DriverAgentMutex.Lock()
				v.DriverAgents[d.Id] = d
				v.DriverAgentMutex.Unlock()
				d.E = v
				//fmt.Printf("[Driver %d]Migrating to %d\n", d.Id, v.Id)
				return true
			}
		}
	}
	return false
}

func (d *DriverAgent) ProcessTask3() {
	<-d.S.StartDrivers // close(StartDrivers) to start this function
	if d.E.S.DriverParameters.TravellingMode == "node" {
		go d.Drive3()
	} else {
		go d.DriveDistance()
	}
	go d.RunComputeRegret()
	sdw := &StartDestinationWaypoint{}  // driver current position to task start
	sdw2 := &StartDestinationWaypoint{} // task start to task end
	tick := time.Tick(d.E.S.MasterSpeed * time.Millisecond)
	for {
		select {
		case <-d.E.Stop:
			//fmt.Println("Driver " + strconv.Itoa(d.Id) + " ended his day")
			return
		case <-tick:
			//fmt.Printf("[Driver %d - Env %d]%v \n", d.Id, d.E.Id, d.Status.String())
			switch d.Status {
			case Roaming:
				//fmt.Printf("[Driver %d - Env %d]%v \n", d.Id, d.E.Id, d.Status.String())
			case Allocating:
				//fmt.Printf("[Driver %d - Env %d]%v \n", d.Id, d.E.Id, d.Status.String())
				select {
				case t := <-d.Request:
					// fmt.Printf("Task address @ during matching: %p\n", &t)
					if t.Task.Id != "null" {
						d.Status = Matching
						d.PendingTask = t.Task
						d.CurrentTask = t.Task
						sdw = &t.StartDestinationWaypoint
						sdw2 = &t.StartDestinationWaypoint2
						//fmt.Printf("Driver %d got Task %v! \n", d.Id, t.Task.Id)
					} else {
						d.Status = Roaming
						//fmt.Printf("Driver %d did not get any tasks \n", d.Id)
					}
				default:

				}
			case Matching:
				if d.AcceptRide() == true {
					d.CurrentTask = d.PendingTask
					d.PendingTask = Task{Id: "null", FinalValue: 0.00} // bad way...
					d.Status = Fetching

					d.E.S.Environments[d.CurrentTask.EnvironmentId].Dispatcher.Response <- DriverMatchingResult{Accept: true, Id: d.Id}
					//d.E.S.Environments[d.CurrentTask.EnvironmentId].TaskMutex.Lock()
					d.E.S.Environments[d.CurrentTask.EnvironmentId].Tasks[d.CurrentTask.Id].WaitStart = d.E.S.SimulationTime
					//d.E.S.Environments[d.CurrentTask.EnvironmentId].TaskMutex.Unlock()
					d.DestinationLocation = d.CurrentTask.StartCoordinate

					d.Recieve <- Message{StartDestinationWaypoint: *sdw}
				} else {
					// send Tasks to TaskQueue
					d.E.TaskQueue <- d.CurrentTask
					d.PendingTask = Task{Id: "null", FinalValue: 0.00}
					d.Status = Roaming
				}
				//fmt.Printf("[Driver %d - Env %d]%v \n", d.Id, d.E.Id, d.Status.String())
			case Fetching:

				// virus

				reached, ok := d.ReachTaskPosition()
				if ok == true && reached == true {
					// driver has arrived at task location
					x := d.E.S.SimulationTime
					d.CurrentTask.WaitEnd = x

					//d.E.S.Environments[d.CurrentTask.EnvironmentId].TaskMutex.Lock()
					d.E.S.Environments[d.CurrentTask.EnvironmentId].Tasks[d.CurrentTask.Id].WaitEnd = x
					//d.E.S.Environments[d.CurrentTask.EnvironmentId].TaskMutex.Unlock()

					d.Status = Travelling
					d.DestinationLocation = d.CurrentTask.EndCoordinate

					d.Recieve <- Message{StartDestinationWaypoint: *sdw2}
				}
			case Travelling:
				// virus

				reached, ok := d.ReachTaskDestination()
				if ok == true && reached == true {
					// driver has completed the task
					x := d.E.S.SimulationTime
					d.CurrentTask.TaskEnded = x

					//d.E.S.Environments[d.CurrentTask.EnvironmentId].TaskMutex.Lock()
					d.E.S.Environments[d.CurrentTask.EnvironmentId].Tasks[d.CurrentTask.Id].TaskEnded = x
					//d.E.S.Environments[d.CurrentTask.EnvironmentId].TaskMutex.Unlock()
					d.E.TotalTaskCompleted++
					d.E.FinishQueue <- d.CurrentTask // put into FinishQueue
					d.CompleteTask()
					d.ComputeFatigue()
					d.ComputeMotivation()
					d.Status = Roaming
					d.DriveToNextPoint4()
				}
			}
			// migrate to another environment

			d.Migrate()

		}
	}
	// fmt.Printf("[Driver %d - Env %d]Driver agent function finished\n", d.Id, d.E.Id)
	log.Fatalf("[Driver %d]Process Ended\n", d.Id)
}

func (d *DriverAgent) Drive3() {
	// 50 miliseconds
	tick := time.Tick(time.Duration(d.E.S.DriverParameters.TravelInterval) * time.Millisecond)
	start := d.S.RN.GetRandomLocation()
	d.CurrentLocation = start
	for {
		//K:
	K:
		select {
		case <-d.E.S.Stop:
			return
		case <-tick:
			select {
			case m := <-d.Recieve:
				d.CurrentLocation = m.StartDestinationWaypoint.StartLocation
				d.DestinationLocation = m.StartDestinationWaypoint.DestinationLocation
				d.Waypoint = m.StartDestinationWaypoint.Waypoint
				break K
			default:
				d.DriveToNextPoint4()
				d.InteractVirus(&d.CurrentTask) // virus function
			}

		}
	}
	panic("[Driver]unreachable\n")
}

func (d *DriverAgent) DriveDistance() {
	start := d.S.RN.GetRandomLocation()
	d.CurrentLocation = start
	for {
		select {
		case <-d.E.S.Stop:
			return
		default:
			select {
			case m := <-d.Recieve:
				d.CurrentLocation = m.StartDestinationWaypoint.StartLocation
				d.DestinationLocation = m.StartDestinationWaypoint.DestinationLocation
				d.Waypoint = m.StartDestinationWaypoint.Waypoint
				//break K
			default:
				d.DriveToNextPointDistance()
				//d.InteractVirus(&d.CurrentTask) // virus function
			}

		}
	}
	panic("[Driver]unreachable\n")
}

func (d *DriverAgent) GetWayPoint() {
	fmt.Printf("[Driver %d - %v]Getting waypoint\n", d.Id, d.Status)
	startTime := time.Now()
	start, end, distance, waypoints := d.S.RN.GetWaypoint(d.CurrentLocation, d.DestinationLocation)
	if distance == math.Inf(1) {
		fmt.Printf("start:%v end:%v distance: %v waypoints: %v\n", start, end, distance, len(waypoints))
		panic("waypoint is 0")
	}

	switch x := len(waypoints); {
	case x > 1:
		waypoints = waypoints[1:]
	case x == 1:
		waypoints = waypoints
	case x == 0:
		//fmt.Printf("start:%v end:%v distance: %v waypoints: %v\n", start, end, distance, len(waypoints))
		panic("waypoint is 0")
	}

	m := &Message{
		StartDestinationWaypoint: StartDestinationWaypoint{
			StartLocation:       start,
			DestinationLocation: end,
			Waypoint:            waypoints,
		},
	}

	elapsed := time.Since(startTime)
	d.Recieve <- *m
	fmt.Printf("[Driver %d - %v]Get waypoint: %s \n", d.Id, d.Status, elapsed)
}

func (d *DriverAgent) DriveToNextPoint4() {
	switch noOfWaypoints := len(d.Waypoint); {
	case noOfWaypoints >= 2:
		d.CurrentLocation = d.Waypoint[0]
		d.Waypoint = d.Waypoint[1:]
	case noOfWaypoints == 1:
		d.CurrentLocation = d.Waypoint[0]
		d.Waypoint = d.Waypoint[:0]
	case noOfWaypoints == 0:
		// TODO: Keep the nextLocation. When route is created, go the past random nextLocation, then the actual waypoint
		if d.Status == Roaming || d.Status == Matching || d.Status == Allocating {
			//time.Sleep(20 * time.Millisecond)
			nextLocation := d.S.RN.GetNextPoint(d.CurrentLocation)
			d.CurrentLocation = nextLocation
		}
	}
}

func (d *DriverAgent) DriveToNextPointDistance() {
	switch noOfWaypoints := len(d.Waypoint); {
	case noOfWaypoints >= 2:
		d.CurrentLocation = d.Waypoint[0]
		distance := CalculateDistance(d.Waypoint[0], d.Waypoint[1])
		timeInSeconds := caluclateTime(distance, d.E.S.DriverParameters.SpeedKmPerHour)
		updatedTime := timeInSeconds * d.S.TickerTime * 2
		time.Sleep(time.Duration(updatedTime) * time.Millisecond)
		d.CurrentLocation = d.Waypoint[0]
		d.Waypoint = d.Waypoint[1:]
	case noOfWaypoints == 1:
		distance := CalculateDistance(d.CurrentLocation, d.Waypoint[0])
		timeInSeconds := caluclateTime(distance, d.E.S.DriverParameters.SpeedKmPerHour)
		updatedTime := timeInSeconds * d.S.TickerTime * 2
		time.Sleep(time.Duration(updatedTime) * time.Millisecond)
		d.CurrentLocation = d.Waypoint[0]
		d.Waypoint = d.Waypoint[:0]
	case noOfWaypoints == 0:
		// TODO: Keep the nextLocation. When route is created, go the past random nextLocation, then the actual waypoint
		if d.Status == Roaming || d.Status == Matching || d.Status == Allocating {
			nextLocation := d.S.RN.GetNextPoint(d.CurrentLocation)

			distance := CalculateDistance(d.CurrentLocation, nextLocation)
			timeInSeconds := caluclateTime(distance, d.E.S.DriverParameters.SpeedKmPerHour)
			updatedTime := timeInSeconds * 200
			time.Sleep(time.Duration(updatedTime) * time.Millisecond)

			d.CurrentLocation = nextLocation
		}
	}
}

func (d *DriverAgent) DriveRandom() {
	_, _, waypoints := d.S.RN.GetEndWaypoint(d.CurrentLocation)
	// d.Waypoint = waypoints
	fmt.Printf("[Driver %v]Sending Random Waypoints: %v\n", d.Id, len(waypoints))
	d.RecieveNewRoute <- waypoints
}

// Accept rate
func (d *DriverAgent) AcceptRide() bool {
	return true
}

func (d *DriverAgent) ReachTaskPosition() (bool, bool) {
	//fmt.Printf("[Driver %d - Env %d]Checking whether reach passenger \n", d.Id, d.E.Id)
	if d.CurrentLocation == d.CurrentTask.StartCoordinate {
		return true, true
	} else {
		return false, true
	}
}

func (d *DriverAgent) ReachTaskDestination() (bool, bool) {
	//fmt.Printf("[Driver %d - Env %d]Checking whether reach passenger's destination \n", d.Id, d.E.Id)
	if d.CurrentLocation == d.CurrentTask.EndCoordinate {
		return true, true
	} else {
		return false, true
	}
}

// After driver completed a task, update rating, taskscompleted, fatigue and motivation
func (d *DriverAgent) CompleteTask() (bool, bool) {

	d.TotalEarnings += d.CurrentTask.FinalValue

	d.CurrentTask.ComputeRating(d.S) //

	d.TaskHistory = append(d.TaskHistory, d.CurrentTask) // add current task to taskhistory since the driver agent has already completed the task
	d.TasksCompleted += d.TasksCompleted + 1             // increment count of TaskCompleted
	d.RecomputeReputation()                              // recompute reputation of driver

	d.CurrentTask = Task{Id: "null", FinalValue: 0.00}

	return true, true
}

func (d *DriverAgent) RecomputeReputation() {
	rating := 0.00
	for _, t := range d.TaskHistory {
		rating = rating + t.RatingGiven
	}
	d.Reputation = rating / float64(len(d.TaskHistory))
}

func (d *DriverAgent) ComputeFatigue() {
	d.Fatigue = d.Fatigue + 1
}

func (d *DriverAgent) ComputeMotivation() {
	d.Motivation = d.Motivation
}

//

func (d *DriverAgent) RunComputeRegret() {
	// d.E.S.UpdateMapSpeed
	x := time.Tick(d.E.S.RegretTickerTime * time.Millisecond)
	for {
		select {
		case <-d.E.S.Stop:
			return
		case <-x:
			d.ComputeRegret()
		}
	}
}

func (d *DriverAgent) ComputeRegret() {
	averageValueOfOrders := d.S.ComputeAverageValue(d)
	computedRegret := d.Regret - float64(d.CurrentTask.FinalValue) + averageValueOfOrders

	if computedRegret < 0 {
		d.Regret = 0
	} else {
		d.Regret = computedRegret
	}

}

func (d *DriverAgent) GetRankingIndex(mmRepFat *[2][2]float64) float64 {
	var rankingIndex float64 = 0

	reputation := GetNormalizedValue(mmRepFat[0], d.Reputation)
	fatigue := GetNormalizedValue(mmRepFat[1], d.Fatigue)

	rankingIndex = (d.Motivation*(reputation-fatigue) + d.Regret)
	return rankingIndex
}

func (d *DriverAgent) GetRawRankingIndex() float64 {
	rankingIndex := (d.Motivation*(d.Reputation-d.Fatigue) + d.Regret)
	return rankingIndex
}

func GetNormalizedValue(mm [2]float64, value float64) float64 {
	min := mm[0]
	max := mm[1]

	// if max and min happen to be same, means all drivers have the same rating. Hence, this value meant nothing in the resultant equation
	if min == max {
		return 1
	}

	return ((value - min) / (max - min))
}

func (d *DriverAgent) InteractVirus(t *Task) {
	currentTask := d.CurrentTask
	if currentTask.Id != "null" || currentTask.Id != "nullopenpathpipepkg/pop3quitread" {
		if d.Virus == None && t.Virus == None {
			return
		} else if d.Virus != None && d.Status == Travelling && t.Virus == None { // Driver spreading virus to passengers
			if SpreadVirus(d.S.VirusParameters, d.Mask) && d.Virus.InAir(d.S.VirusParameters) && RecieveVirus(d.S.VirusParameters, t.Mask) {
				fmt.Printf("current task - env: %v, id: %v\n", currentTask.EnvironmentId, currentTask.Id)
				d.E.S.Environments[currentTask.EnvironmentId].Tasks[currentTask.Id].Virus = Mild
				t.Virus = Mild
				d.E.DriversToTasks++
			}
		} else if d.Virus == None && d.Status == Travelling && t.Virus != None { // Task spreading virus to passengers
			if SpreadVirus(d.S.VirusParameters, t.Mask) && t.Virus.InAir(d.S.VirusParameters) && RecieveVirus(d.S.VirusParameters, d.Mask) {
				d.Virus = Mild
				d.E.TasksToDrivers++
			}
		}
	}

	if d.Virus != None {
		d.Virus = d.Virus.Evolve(d.S.VirusParameters)
	}

}

func (d *DriverAgent) GetRankingIndexParams(
	motivationMinMax [2]float64,
	reputationMinMax [2]float64,
	fatigueMinMax [2]float64,
	regretMinMax [2]float64,
	motivation float64,
	reputation float64,
	fatigue float64,
	regret float64,
) float64 {
	var rankingIndex float64 = 0

	newMotivation := GetNormalizedValue(motivationMinMax, d.Reputation)
	newReputation := GetNormalizedValue(reputationMinMax, reputation)
	newFatigue := GetNormalizedValue(fatigueMinMax, fatigue)
	newRegret := GetNormalizedValue(regretMinMax, regret)

	rankingIndex = (newMotivation*(newReputation-newFatigue) + newRegret)
	return rankingIndex
}
