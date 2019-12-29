package dispatchsim

import (
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

type Environment struct {
	S                    *Simulation
	Id                   int
	DriverAgents         map[int]*DriverAgent
	NoOfIntialDrivers    int
	IncomingDriversQueue chan DriverAgent // TODO: for migrating drivers
	Tasks                map[string]*Task // store all tasks
	TaskQueue            chan Task
	FinishQueue          chan Task
	TotalTasks           int
	MasterSpeed          time.Duration
	Quit                 chan struct{}
	WaitGroup            *sync.WaitGroup // for ending simulation
}

func SetupEnvironment(s *Simulation, id int, noOfDrivers int, generateDrivers bool, generateTasks bool) Environment {
	return Environment{
		S:                    s,
		Id:                   id,
		DriverAgents:         make(map[int]*DriverAgent),
		NoOfIntialDrivers:    noOfDrivers,
		IncomingDriversQueue: make(chan DriverAgent, 1000),
		Tasks:                make(map[string]*Task), // id -> task
		TaskQueue:            make(chan Task, 10000),
		FinishQueue:          make(chan Task, 10000),
		TotalTasks:           0,
		MasterSpeed:          100,
		Quit:                 make(chan struct{}), // for stopping dispatcher and drivers
	}
}

// generate tasks
func (e *Environment) GiveTask(o Order) {
	task := CreateTaskFromOrder(o)
	fmt.Printf("[Environment %d]New Task Generated! - Task %v \n", e.Id, task.Id)
	e.TaskQueue <- task
	e.Tasks[task.Id] = &task
	fmt.Printf("Task address @ givetask function: %p \n", &task)
}

// generate new drivers
//TODO: how to get last index no...?
func (e *Environment) GenerateDriver(name string, id int) {
	//e.DriverAgents = append(e.DriverAgents, CreateDriver(id, e))
	if _, ok := e.DriverAgents[id]; !ok {
		driver := CreateDriver(id, e)
		e.DriverAgents[id] = &driver
	} else {
		log.Fatal("Driver exists!")
	}

}

func (e *Environment) Run(startingDriverId int) {
	// for i := 0; i < len(e.DriverAgents); i++ {
	// 	e.DriverAgents[i] = CreateDriver(startingDriverId, e)
	// 	go e.DriverAgents[i].ProcessTask()
	// 	startingDriverId++
	// }

	for i := 0; i < e.NoOfIntialDrivers; i++ {
		driver := CreateDriver(startingDriverId, e)
		e.DriverAgents[startingDriverId] = &driver
		go e.DriverAgents[startingDriverId].ProcessTask2()
		startingDriverId++
	}

	go dispatcher(e)

	// condition to end simulation
	// 	var countTask int = 0
	// End:
	// 	for {
	// 		select {
	// 		case task := <-e.FinishQueue:
	// 			fmt.Printf("[Environment %d]Task %d completed :) \n", e.Id, task.Id)
	// 			countTask++
	// 			fmt.Printf("[Environment %d]Task completed: %d / %d\n", e.Id, countTask, e.TotalTasks)
	// 		}

	// 		if countTask == e.TotalTasks {
	// 			fmt.Printf("[Environment %d]All tasks completed\n", e.Id)
	// 			break End
	// 		}
	// 	}

	// 	// closing simluation
	// 	close(e.Quit) // stop all driver agent goroutines and dispatcher
	// 	fmt.Printf("[Environment %d]Environment ended\n", e.Id)
	// 	e.Stats()
	// 	return
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
	var accumulatedTaskValue = 0
	var totalDriversWithTask = 0

	for _, v := range e.DriverAgents {
		if v.CurrentTask.Id != "null" && v.Status != Roaming {
			fmt.Printf("Driver %d has Task %d with value of %d \n",
				v.Id,
				v.CurrentTask.Id,
				v.CurrentTask.Value,
			)
			accumulatedTaskValue = v.CurrentTask.Value + accumulatedTaskValue
			totalDriversWithTask++
		}
	}
	averageTaskValue := float64(accumulatedTaskValue) / float64(totalDriversWithTask)

	if math.IsNaN(averageTaskValue) {
		return 0
	}
	return averageTaskValue
}
