package dispatchsim

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
)

// Structure of the Environment
type Environment struct {
	S                       *Simulation
	Id                      int
	Dispatcher              *Dispatcher
	Polygon                 orb.Polygon
	PolygonLatLng           []LatLng
	DriverAgents            map[int]*DriverAgent
	DriverAgentMutex        sync.RWMutex
	NoOfIntialDrivers       int
	Tasks                   map[string]*Task // store all tasks
	TasksTimeline           []Task
	TaskMutex               sync.RWMutex
	TaskQueue               chan Task
	FinishQueue             chan Task
	TotalTasks              int
	MasterSpeed             time.Duration
	Stop                    chan struct{}
	IsDistributing          bool
	TasksToDrivers          int // To capture the number of virus spread from tasks(passenger) to drivers
	DriversToTasks          int // To capture the number of virus spread from drivers to tasks(passenger)
	TotalTasksToBeCompleted int // A counter for the total tasks completed needed to be compeleted - each boundary will have a set of tasks
	TotalTaskCompleted      int // A counter for the total tasks completed (passenger that are successfully fetched to their destination)
}

// Note that dispatcher comes at .Run() function
// Intialize of environment
// Please do note that environment == boundary
func SetupEnvironment(s *Simulation, id int, noOfDrivers int, generateDrivers bool, generateTasks bool, p []LatLng) Environment {
	return Environment{
		S:                       s,                              // The envioronment have a reference to Simulation
		Id:                      id,                             // Each environment have an id for tracking purpose
		Polygon:                 ConvertLatLngArrayToPolygon(p), // Each environment
		PolygonLatLng:           p,
		DriverAgents:            make(map[int]*DriverAgent),
		DriverAgentMutex:        sync.RWMutex{},
		NoOfIntialDrivers:       noOfDrivers,
		Tasks:                   make(map[string]*Task), // id -> task
		TasksTimeline:           make([]Task, 0),
		TaskMutex:               sync.RWMutex{},
		TaskQueue:               make(chan Task, 500000),
		FinishQueue:             make(chan Task, 10000),
		TotalTasks:              0,
		MasterSpeed:             100,
		Stop:                    make(chan struct{}), // for stopping dispatcher and drivers
		IsDistributing:          false,
		TasksToDrivers:          0,
		DriversToTasks:          0,
		TotalTasksToBeCompleted: 0,
		TotalTaskCompleted:      0,
	}
}

// Function to "start" the environment(or boundary)
// Each boundary have a dispatcher process and tasks distributor process
// With the additional of the infectious spread among ride sharing system, I have added another process to simulate the mechanism of virus evolving in tasks (or passengers)
func (e *Environment) Run() {
	dis := SetupDispatcher(e)
	e.Dispatcher = &dis
	go dis.dispatcher3(e)
	go e.RunTasksDistributor()
	go e.RunTasksEvolveVirus()

	for {
		select {
		case <-e.FinishQueue:
			fmt.Printf("[Environment]Task: %v/%v\n", e.TotalTaskCompleted, e.TotalTasksToBeCompleted)
			if e.TotalTaskCompleted == e.TotalTasksToBeCompleted {
				close(e.Stop)
				close(e.S.Stop)
				e.S.isRunning = false
				e.S.Recieve <- "[3,1]"
			}
		}
	}
}

// TODO: Support Multiple days. Currently it support one day only.
// This function will act as a scheduler in "dispatching" the tasks to the boundary with respect to the simulation time
// The data used is in sequence with respect to the time of the order starts
// E.g There are 20 orders. First 15 orders begin at 12.01am. Last 5 orders being at 12.05am
// If the simulation time hits 12.01am, 15 orders will appear on the map
// When the simulation time hits 12.05am, another 15 orders will appear on the map
func (e *Environment) RunTasksDistributor() {
	fmt.Printf("[TasksDistributor %v]Awaiting to start\n", e.Id)
	<-e.S.StartDispatchers
	fmt.Printf("[TasksDistributor %v]Started\n", e.Id)
	for range e.S.Ticker {
		//fmt.Printf("[TasksDistributor %v]Time: %v\n", e.Id, e.S.SimulationTime)
		currentTime := e.S.SimulationTime
		sameOrderCount := 0
	F:
		for _, v := range e.TasksTimeline {
			time := ConvertUnixToTimeStamp(v.RideStartTime) // we use RideStartTime as the start of order
			// If the order has the same hour and minutes as the simulation time, release the order to the boundary and let the dispatcher take the order
			// into account for the dispatcher algorithm
			if (currentTime.Hour() == time.Hour()) && (currentTime.Minute() == time.Minute()) {
				e.IsDistributing = true
				e.Tasks[v.Id].Appear = true // appear on the map
				e.TaskQueue <- v            // push task to queue
				sameOrderCount++
			} else {
				break F
			}
		}
		if sameOrderCount > 0 {
			e.TasksTimeline = e.TasksTimeline[sameOrderCount:]
		}
		e.IsDistributing = false

		if len(e.TasksTimeline) == 0 {
			fmt.Printf("[TasksDistributor %v]Finish distributing task to environment\n", e.Id)
			return
		}
	}
}

// Virus version
// Unused function
func (e *Environment) RunTasksDistributorVirus() {
	fmt.Printf("[TasksDistributor %v]Awaiting to start\n", e.Id)
	<-e.S.StartDispatchers
	fmt.Printf("[TasksDistributor %v]Started\n", e.Id)
	for range e.S.Ticker {
		currentTime := e.S.SimulationTime
		sameOrderCount := 0
	F:
		for _, v := range e.TasksTimeline {
			time := ConvertUnixToTimeStamp(v.RideStartTime) // we use RideStartTime as the start of order
			if (currentTime.Hour() == time.Hour()) && (currentTime.Minute() == time.Minute()) {
				e.IsDistributing = true
				e.Tasks[v.Id].Appear = true // appear on the map
				e.TaskQueue <- v            // push task to queue
				sameOrderCount++
			} else {
				break F
			}
		}
		if sameOrderCount > 0 {
			e.TasksTimeline = e.TasksTimeline[sameOrderCount:]
		}
		e.IsDistributing = false

		if len(e.TasksTimeline) == 0 {
			fmt.Printf("[TasksDistributor %v]Finish distributing task to environment\n", e.Id)
			return
		}
	}
}

// For every task in the boundary, lets run the evolve virus mechanism
func (e *Environment) RunTasksEvolveVirus() {
	fmt.Printf("[TasksEvolver %v]Awaiting to start\n", e.Id)
	<-e.S.StartDispatchers
	fmt.Printf("[TasksEvolver %v]Started\n", e.Id)
	tick := time.Tick(200 * time.Millisecond)

	for {
		select {
		case <-tick:
			for _, t := range e.Tasks {
				if t.Virus != None {
					t.Virus = t.Virus.Evolve(e.S.VirusParameters)
				}
			}
		}
	}
}

// Compute average value of orders with similiar rating
// This function is use for driveragent.go
func (e *Environment) ComputeAverageValue(reputation float64) float64 {
	var accumulatedTaskValue float64 = 0
	var totalDriversWithTask = 0

	for _, v := range e.DriverAgents {
		if v.CurrentTask.Id != "null" && v.Status != Roaming {
			fmt.Printf("[ComputeAverageValue] Driver %d has Task %v with value of %d \n",
				v.Id,
				v.CurrentTask.Id,
				v.CurrentTask.FinalValue,
			)
			accumulatedTaskValue = v.CurrentTask.FinalValue + accumulatedTaskValue
			totalDriversWithTask++
		}
	}
	averageTaskValue := float64(accumulatedTaskValue) / float64(totalDriversWithTask)

	if math.IsNaN(averageTaskValue) {
		return 0
	}
	return averageTaskValue
}

// This function check if the point is inside the polygon
func isPointInside(p orb.Polygon, point orb.Point) bool {
	if planar.PolygonContains(p, point) {
		return true
	} else {
		return false
	}
}
