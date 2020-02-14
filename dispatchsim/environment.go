package dispatchsim

import (
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
)

type Environment struct {
	S                 *Simulation
	Id                int
	Dispatcher        *Dispatcher
	Polygon           orb.Polygon
	PolygonLatLng     []LatLng
	DriverAgents      map[int]*DriverAgent
	DriverAgentMutex  sync.RWMutex
	NoOfIntialDrivers int
	Tasks             map[string]*Task // store all tasks
	TasksTimeline     []Task
	TaskMutex         sync.RWMutex
	TaskQueue         chan Task
	FinishQueue       chan Task
	TotalTasks        int
	MasterSpeed       time.Duration
	Quit              chan struct{}
	IsDistributing    bool
	WaitGroup         *sync.WaitGroup // for ending simulation
}

// Note that dispatcher comes at .Run() function
func SetupEnvironment(s *Simulation, id int, noOfDrivers int, generateDrivers bool, generateTasks bool, p []LatLng) Environment {
	return Environment{
		S:                 s,
		Id:                id,
		Polygon:           ConvertLatLngArrayToPolygon(p),
		PolygonLatLng:     p,
		DriverAgents:      make(map[int]*DriverAgent),
		DriverAgentMutex:  sync.RWMutex{},
		NoOfIntialDrivers: noOfDrivers,
		Tasks:             make(map[string]*Task), // id -> task
		TasksTimeline:     make([]Task, 0),
		TaskMutex:         sync.RWMutex{},
		TaskQueue:         make(chan Task, 500000),
		FinishQueue:       make(chan Task, 10000),
		TotalTasks:        0,
		MasterSpeed:       100,
		Quit:              make(chan struct{}), // for stopping dispatcher and drivers
		IsDistributing:    false,
	}
}

// generate tasks
func (e *Environment) GiveTask(o Order) {
	task := CreateTaskFromOrder(o, e)
	fmt.Printf("[Environment %d]New Task Generated! - Task %v \n", e.Id, task.Id)
	e.TaskQueue <- task
	//e.TaskMutex.Lock()
	e.Tasks[task.Id] = &task
	//e.TaskMutex.Unlock()
}

func (e *Environment) Run() {

	dis := SetupDispatcher(e)
	e.Dispatcher = &dis
	go dis.dispatcher3(e)
	go e.RunTasksDistributor()
}

// TODO: Do multiple days.
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
			e.S.SendMessageToClient("Next order is at " + strconv.Itoa(time.Hour()) + ":" + strconv.Itoa(time.Minute()))
			//fmt.Printf("[TasksDistributor %v]Task %v time %v , hour:%v, minute:%v, c.hour:%v ,c.minute:%v\n", e.Id, v.Id, time, time.Hour(), time.Minute(), currentTime.Hour(), currentTime.Minute())
			if (currentTime.Hour() == time.Hour()) && (currentTime.Minute() == time.Minute()) {
				e.IsDistributing = true
				//fmt.Printf("[TasksDistributor %v]Task %v sent to environment at time %v\n", e.Id, v.Id, time)
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

// compute average value of orders with similiar rating
// TODO: similiar rating should be adjustable.... i think...
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

func isPointInside(p orb.Polygon, point orb.Point) bool {
	// for _, v := range s.Environments {
	if planar.PolygonContains(p, point) {
		//fmt.Printf("[isPointInsidePolygon]Point inside in Environment!\n")
		return true
	} else {
		//fmt.Printf("[isPointInsidePolygon]Point not inside in Environment\n")
		return false
	}
}
