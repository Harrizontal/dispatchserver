package dispatchsim

import (
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"
)

type Environment struct {
	DriverAgents []DriverAgent
	TaskQueue    chan Task
	FinishQueue  chan Task
	IntialTasks  int
	MasterSpeed  time.Duration
	Quit         chan struct{}
	WaitGroup    *sync.WaitGroup // for ending simulation
}

// generate tasks
func (e *Environment) GenerateTask(d int) {
	task := CreateTask(d)
	fmt.Println("New Task Generated!")
	e.TaskQueue <- task
}

// generate new drivers
//TODO: how to get last index no...?
func (e *Environment) GenerateDriver(name string, id int) {
	e.DriverAgents = append(e.DriverAgents,
		CreateDriver(id, e))
}

// func (e *Environment) GenerateCustomer(name string, id int) {
// 	e.CustomerAgents = append(e.CustomerAgents,
// 		CustomerAgent{Name: name, Id: id})
// }

func (e *Environment) Run() {
	for i := 0; i < len(e.DriverAgents); i++ {
		e.DriverAgents[i] = CreateDriver(i, e)
		go e.DriverAgents[i].ProcessTask()
	}
	go dispatcher(e)

	for i := 0; i < e.IntialTasks; i++ {
		e.GenerateTask(i)
	}

	// condition to end simulation
	var countTask int = 0
	for {
		select {
		case x := <-e.FinishQueue:
			fmt.Println("Task " + strconv.Itoa(x.Id) + " completed :)")
			countTask++
			fmt.Println("Task completed: " + strconv.Itoa(countTask) + "/" + strconv.Itoa(e.IntialTasks))
		}

		if countTask == e.IntialTasks {
			fmt.Println("[Simulation] All tasks completed")
			break
		}
	}

	// closing simluation
	close(e.Quit)
	e.WaitGroup.Done()
	fmt.Println("[Simulation] Simulation end")
	return
}

func SetupEnvironment(noOfDrivers int, noOfTask int, wg *sync.WaitGroup) Environment {
	return Environment{
		DriverAgents: make([]DriverAgent, noOfDrivers),
		TaskQueue:    make(chan Task, 10000),
		FinishQueue:  make(chan Task, 10000),
		IntialTasks:  noOfTask,
		MasterSpeed:  100,
		Quit:         make(chan struct{}), // for stopping dispatcher and drivers
		WaitGroup:    wg,                  // for stopping simulation
	}
}

func (e *Environment) Stats() {
	fmt.Println("Stats:")
	for i := 0; i < len(e.DriverAgents); i++ {
		fmt.Printf("Driver %d's Stats - TasksCompleted: %d, Reputation: %f, Fatigue: %f, Motivation %f, Regret %f\n",
			i,
			e.DriverAgents[i].TasksCompleted,
			e.DriverAgents[i].Reputation,
			e.DriverAgents[i].Fatigue,
			e.DriverAgents[i].Motivation,
			e.DriverAgents[i].Regret)
	}
}

// compute average value of orders with similiar rating
// TODO: similiar rating should be adjustable.... i think...
func (e *Environment) ComputeAverageValue(reputation float64) float64 {
	var accumulatedTaskValue = 0
	var totalDriversWithTask = 0
	for i := 0; i < len(e.DriverAgents); i++ {
		if e.DriverAgents[i].CurrentTask.Id >= 0 &&
			e.DriverAgents[i].Status != Roaming { // bad way to compare whether currenttask not null
			// Check with zi chao/prof yu on calculating average value of orders among drivers (all? those currently have? )
			fmt.Printf("Driver %d has Task %d with value of %d \n",
				e.DriverAgents[i].Id,
				e.DriverAgents[i].CurrentTask.Id,
				e.DriverAgents[i].CurrentTask.Value)
			accumulatedTaskValue = e.DriverAgents[i].CurrentTask.Value + accumulatedTaskValue
			totalDriversWithTask++
		}
	}

	averageTaskValue := float64(accumulatedTaskValue) / float64(totalDriversWithTask)

	if math.IsNaN(averageTaskValue) {
		return 0
	}
	return averageTaskValue
}
