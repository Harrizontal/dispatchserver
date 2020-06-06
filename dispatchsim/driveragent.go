package dispatchsim

import (
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/paulmach/orb"
)

// Sort of an "enum" for DriverStatus
type DriverStatus int

// There are 5 status for drivers
// Roaming - Drivers roam around the map and waiting for tasks assigned by the dispatcher
// Matching - Drivers is in the midst of accepting the tasks (currently - all driver will 100% accept the tasks)
// Fetching (it is known as Picking up in the user interface) - Drivers is PICKING UP the passenger
// Travelling (also known as Fetching in the user interface) - The passenger is with the driver now. Driver is fetching the passenger to the destination
// Allocating - The driver is being accounted in the dispatcher algorithm. When the driver has a status of "Allocating", other dispatchers (there could be intercepting boundaries defined by the user) cannot take into the account of this driver
// First 4 is visible in the visualization tool, but not Allocating as it is sort of a hidden status.
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
	S                   *Simulation   // The driveragent belongs to the simulation
	E                   *Environment  // The driveragent belongs to an environment (or boundary)
	Id                  int           // Each driver holds a id
	Name                string        // Just name of the driver.
	CurrentLocation     LatLng        // Current location of the driver
	Waypoint            []LatLng      // An array of Latlng for drivers to go to.
	NextLocation        LatLng        // Next Latlng for driver to go
	DestinationLocation LatLng        // Destination of the driver (typically is updated when the driver holds a task)
	PendingTask         Task          // store task during matching
	CurrentTask         Task          // store task when fetching/travelling
	Status              DriverStatus  // Status of the driver
	TotalEarnings       float64       // Total earnings of driver earned in the simulation
	Request             chan Message  // Request
	Recieve             chan Message  // Receive
	Recieve2            chan int      // Recieve
	RecieveNewRoute     chan []LatLng // Recieve a new array of LatLng
	RequestNewRoute     bool          // Unused variable
	Send                chan string   // Send
	TasksCompleted      int           // Number of tasks completed for the driver
	TaskHistory         []Task        // Track the history of tasks completed
	Motivation          float64       // Motivation variable
	Reputation          float64       // Reputation variable
	Fatigue             float64       // Fatigue variable
	Regret              float64       // Regret variable
	RankingIndex        float64       // RankingIndex variable
	ChangeDestination   chan string
	StartDriving        time.Time    // Time started for this driver
	EndDriving          time.Time    // Time ended for this driver
	Valid               bool         // Valid is important variable. If valid == true, the dispatcher can take into account of this driver while executing the dispatcher algorithm. If valid == false, the driver is deemed as an islander (stuck in an island)
	Virus               Virus        // Virus has 3 levels.
	Mask                bool         // If true, driver got wear mask. Else, driver doesnt wear mask
	DriverMutex         sync.RWMutex // Mutex
}

// Function to create multiple drivers in the Simluation (random area)
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

// Function to intitalize one driver agent
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

// Virus version
// Function to create multiple drivers in the Simluation (random area)
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

// Virus version
// Function to intitalize one driver agent
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

// Migrating to another environment (or boundary)
// E.g If driver is in environment 1, and a few seconds later, the driver is in envrionment 2.
// Driver have to migrate to environment 2 for the dispatcher in environment 2 to assign tasks to driver
func (d *DriverAgent) Migrate() bool {
	// if the driver's location is in the not same
	cl := d.CurrentLocation
	point := orb.Point{cl.Lng, cl.Lat}
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

				return true
			}
		}
	}
	return false
}

func (d *DriverAgent) ProcessTask3() {
	<-d.S.StartDrivers // close(StartDrivers) to start this function
	// There are two travelling mode for the driver. Travel by node and travel by distance
	// Select travel by distance for a more realistic approach for the simulation (could take longer for the simulation to complete)
	// Select travel by node for a way to visualization how the virus spread or complete tasks
	if d.E.S.DriverParameters.TravellingMode == "node" {
		go d.DriveNode()
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
			return
		case <-tick:
			switch d.Status {
			case Roaming:
			case Allocating:
				// If the driver has a status of Allocating, the driver should be expecing a task
				// If the driver is not expecting a task (recieves a null task), driver status will go back to Roaming
				select {
				case t := <-d.Request:
					// fmt.Printf("Task address @ during matching: %p\n", &t)
					if t.Task.Id != "null" {
						d.Status = Matching
						d.PendingTask = t.Task
						d.CurrentTask = t.Task
						sdw = &t.StartDestinationWaypoint
						sdw2 = &t.StartDestinationWaypoint2
					} else {
						d.Status = Roaming
					}
				default:

				}
			case Matching:
				// Driver accepts the tasks
				if d.AcceptRide() == true {
					d.CurrentTask = d.PendingTask
					d.PendingTask = Task{Id: "null", FinalValue: 0.00} // bad way...
					d.Status = Fetching

					// Tell the dispatcher that the driver has accepted the task
					// Obselete function
					d.E.S.Environments[d.CurrentTask.EnvironmentId].Dispatcher.Response <- DriverMatchingResult{Accept: true, Id: d.Id}

					d.E.S.Environments[d.CurrentTask.EnvironmentId].Tasks[d.CurrentTask.Id].WaitStart = d.E.S.SimulationTime

					d.DestinationLocation = d.CurrentTask.StartCoordinate

					d.Recieve <- Message{StartDestinationWaypoint: *sdw}
				} else {

					d.E.TaskQueue <- d.CurrentTask
					d.PendingTask = Task{Id: "null", FinalValue: 0.00}
					d.Status = Roaming
				}

			case Fetching:
				// Note: Fetching is equalivant to picking up the passenger

				// if the driver has reach the start location of the driver
				reached, ok := d.ReachTaskPosition()
				if ok == true && reached == true {
					// driver has arrived at task location
					x := d.E.S.SimulationTime
					d.CurrentTask.WaitEnd = x

					d.E.S.Environments[d.CurrentTask.EnvironmentId].Tasks[d.CurrentTask.Id].WaitEnd = x

					d.Status = Travelling                               // Set to Travelling
					d.DestinationLocation = d.CurrentTask.EndCoordinate // Set end coordinate to driver

					d.Recieve <- Message{StartDestinationWaypoint: *sdw2}
				}
			case Travelling:
				// Note: Travelling is equalivant to fetching the passenger to the destination

				// If the driver has reach the destination
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
					d.DriveToNextPoint()
				}
			}
			// migrate to another boundary (provided if the driver is in another boundary)
			d.Migrate()

		}
	}
	log.Fatalf("[Driver %d]Process Ended\n", d.Id)
}

func (d *DriverAgent) DriveNode() {
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
				d.DriveToNextPoint()
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

// If there still Latlng in the waypoint, let the driver travel to the LatLng in the wapoint first.
// If there is no more waypoint, driver can travel to a random node
func (d *DriverAgent) DriveToNextPoint() {
	switch noOfWaypoints := len(d.Waypoint); {
	case noOfWaypoints >= 2:
		d.CurrentLocation = d.Waypoint[0]
		d.Waypoint = d.Waypoint[1:]
	case noOfWaypoints == 1:
		d.CurrentLocation = d.Waypoint[0]
		d.Waypoint = d.Waypoint[:0]
	case noOfWaypoints == 0:
		if d.Status == Roaming || d.Status == Matching || d.Status == Allocating {
			// Go a random node (but next to it)
			// TODO: Design a better random driving mechanism but I am limited to CPU usage
			nextLocation := d.S.RN.GetNextPoint(d.CurrentLocation)
			d.CurrentLocation = nextLocation
		}
	}
}

// Similar to DriveToNextPoint but accounting the distance.
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

// Function for driver to driver randomly
func (d *DriverAgent) DriveRandom() {
	_, _, waypoints := d.S.RN.GetEndWaypoint(d.CurrentLocation)
	// d.Waypoint = waypoints
	fmt.Printf("[Driver %v]Sending Random Waypoints: %v\n", d.Id, len(waypoints))
	d.RecieveNewRoute <- waypoints
}

// Accept rate
// Currently is 100%
func (d *DriverAgent) AcceptRide() bool {
	return true
}

// Check if the driver has reach the start location of the Task
func (d *DriverAgent) ReachTaskPosition() (bool, bool) {
	if d.CurrentLocation == d.CurrentTask.StartCoordinate {
		return true, true
	} else {
		return false, true
	}
}

// Check if driver has reach the destination
func (d *DriverAgent) ReachTaskDestination() (bool, bool) {
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

// Calculate the reputation of the driver once the task is completed
// The task will give a rating to the driver (could be random or predefined from 0 to 5 - user can set at the visualization tool)
func (d *DriverAgent) RecomputeReputation() {
	rating := 0.00
	for _, t := range d.TaskHistory {
		rating = rating + t.RatingGiven
	}
	d.Reputation = rating / float64(len(d.TaskHistory))
}

// For every successful tasks completed, fatigue increases
func (d *DriverAgent) ComputeFatigue() {
	d.Fatigue = d.Fatigue + 1
}

// Motivation stays the same for now
func (d *DriverAgent) ComputeMotivation() {
	d.Motivation = d.Motivation
}

// Compute the regret of the driver
// The computation of regret is a process
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

// Calculate the ranking index of the driver
func (d *DriverAgent) GetRankingIndex(mmRepFat *[2][2]float64) float64 {
	var rankingIndex float64 = 0

	reputation := GetNormalizedValue(mmRepFat[0], d.Reputation)
	fatigue := GetNormalizedValue(mmRepFat[1], d.Fatigue)

	rankingIndex = (d.Motivation*(reputation-fatigue) + d.Regret)
	return rankingIndex
}

// Calculate the RAW ranking index (without normalization)
func (d *DriverAgent) GetRawRankingIndex() float64 {
	rankingIndex := (d.Motivation*(d.Reputation-d.Fatigue) + d.Regret)
	return rankingIndex
}

// Utility function to get the normalized value based on the min and max
func GetNormalizedValue(mm [2]float64, value float64) float64 {
	min := mm[0]
	max := mm[1]

	// if max and min happen to be same, means all drivers have the same rating. Hence, this value meant nothing in the resultant equation
	if min == max {
		return 1
	}

	return ((value - min) / (max - min))
}

// Virus mechanism in driver agent
// If the driver has passenger, it will execute the spread mechanism
// Whether it is spread successfully is dependent on the probability set by the user
// If the driver has virus, it also will execute the evolve mechanism
// Whether it is evolved successfully is dependent on the probability set by the user too.
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

// Calculate the ranking index based on the normalized motivation, reputation, fatigue and regret.
// This is cruicial in deciding the task is allocated to the driver
// Higher the index, the higher the value in task allocated to the driver
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
