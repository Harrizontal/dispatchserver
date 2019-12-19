package dispatchsim

import (
	"fmt"
	"math"
	"sync"
	"time"
)

type Environment struct {
	S                    *Simulation
	Id                   int
	DriverAgents         []DriverAgent
	IncomingDriversQueue chan DriverAgent // TODO: for migrating drivers
	TaskQueue            chan Task
	FinishQueue          chan Task
	TotalTasks           int
	MasterSpeed          time.Duration
	Quit                 chan struct{}
	WaitGroup            *sync.WaitGroup // for ending simulation
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

func (e *Environment) Run(startingDriverId int) {
	for i := 0; i < len(e.DriverAgents); i++ {
		e.DriverAgents[i] = CreateDriver(startingDriverId, e)
		go e.DriverAgents[i].ProcessTask()
		startingDriverId++
	}

	go dispatcher(e)

	// condition to end simulation
	var countTask int = 0
End:
	for {
		select {
		case task := <-e.FinishQueue:
			fmt.Printf("[Environment %d]Task %d completed :) \n", e.Id, task.Id)
			countTask++
			fmt.Printf("[Environment %d]Task completed: %d / %d\n", e.Id, countTask, e.TotalTasks)
		}

		if countTask == e.TotalTasks {
			fmt.Printf("[Environment %d]All tasks completed\n", e.Id)
			break End
		}
	}

	// closing simluation
	close(e.Quit) // stop all driver agent goroutines and dispatcher
	fmt.Printf("[Environment %d]Environment ended\n", e.Id)
	return
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
