package dispatchsim

import (
	"fmt"
	"math"
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
	TaskMutex         sync.RWMutex
	TaskQueue         chan Task
	FinishQueue       chan Task
	TotalTasks        int
	MasterSpeed       time.Duration
	Quit              chan struct{}
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
		TaskMutex:         sync.RWMutex{},
		TaskQueue:         make(chan Task, 10000),
		FinishQueue:       make(chan Task, 10000),
		TotalTasks:        0,
		MasterSpeed:       100,
		Quit:              make(chan struct{}), // for stopping dispatcher and drivers
	}
}

// generate tasks
func (e *Environment) GiveTask(o Order) {
	task := CreateTaskFromOrder(o, e)
	fmt.Printf("[Environment %d]New Task Generated! - Task %v \n", e.Id, task.Id)
	e.TaskQueue <- task
	e.TaskMutex.Lock()
	e.Tasks[task.Id] = &task
	e.TaskMutex.Unlock()
}

// generate new drivers
//TODO: how to get last index no...?
// func (e *Environment) GenerateDriver(name string, id int) {
// 	//e.DriverAgents = append(e.DriverAgents, CreateDriver(id, e))
// 	if _, ok := e.DriverAgents[id]; !ok {
// 		driver := CreateDriver(id, e)
// 		e.DriverAgents[id] = &driver
// 	} else {
// 		log.Fatal("Driver exists!")
// 	}

// }

func (e *Environment) Run() {

	dis := SetupDispatcher(e)
	e.Dispatcher = &dis
	go dis.dispatcher(e)

}

func (e *Environment) Stats() {
	fmt.Println("Stats:")
	for k, v := range e.DriverAgents {
		fmt.Printf("Driver %d's Stats - TasksCompleted: %d, Reputation: %f, Fatigue: %f, Motivation %f, Regret %f, Ranking Index: %f\n",
			k,
			v.TasksCompleted,
			v.Reputation,
			v.Fatigue,
			v.Motivation,
			v.Regret,
			v.GetRankingIndex())
	}
	// for i := 0; i < len(e.DriverAgents); i++ {
	// 	fmt.Printf("Driver %d's Stats - TasksCompleted: %d, Reputation: %f, Fatigue: %f, Motivation %f, Regret %f\n",
	// 		i,
	// 		e.DriverAgents[i].TasksCompleted,
	// 		e.DriverAgents[i].Reputation,
	// 		e.DriverAgents[i].Fatigue,
	// 		e.DriverAgents[i].Motivation,
	// 		e.DriverAgents[i].Regret)
	// }
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
