package dispatchsim

import (
	"fmt"
	"sort"
	"time"
)

// Implement algorithm here
func (dis *Dispatcher) dispatcher(e *Environment) {
	tick := time.Tick(time.Duration(e.S.DispatcherParameters.DispatchInterval) * time.Millisecond) // TODO: master timer

	for {
		select {
		case <-e.Quit:
			return
		case <-tick:

			noOfRoamingDrivers := 0
			var roamingDrivers = make([]*DriverAgent, 0)
			e.DriverAgentMutex.Lock()
			for _, v := range e.DriverAgents {
				if v.Status == Roaming {
					roamingDrivers = append(roamingDrivers, v)
					noOfRoamingDrivers++
					v.Status = Allocating // change roaming to allocating to prevent double task allocation when migrating
				}
			}
			e.DriverAgentMutex.Unlock()
			//TODO: Duplicate code?
			sort.SliceStable(roamingDrivers, func(i, j int) bool {
				return roamingDrivers[i].GetRankingIndex() > roamingDrivers[j].GetRankingIndex()
			})

			//fmt.Printf("NoOfRoamingDrivers: %d\n", noOfRoamingDrivers)

			if noOfRoamingDrivers > 0 {
				//fmt.Println("[Dispatcher]Start allocation")
				tasks := GetValuableTasks(noOfRoamingDrivers, e.TaskQueue)
				if len(tasks) > 0 {
					fmt.Printf("[Dispatcher %d]===Assigning task(s) to roaming driver(s)===\n", e.Id)
					for i := 0; i < len(tasks); i++ {
						//fmt.Printf("[Dispatcher]Task %d with value of %d is to be allocated \n", tasks[i].Id, tasks[i].Value)
					}

					// Sort drivers' Driver Ranking Index in descending order!
					sort.SliceStable(roamingDrivers, func(i, j int) bool {
						return roamingDrivers[i].GetRankingIndex() > roamingDrivers[j].GetRankingIndex()
					})

					// for printing - can delete!
					// for i := 0; i < len(roamingDrivers); i++ {
					// 	fmt.Printf("[Dispatcher]Roaming Driver %d with movtivation %f, reputation %f , fatigue %f and regret %f. Ranking Index: %f\n",
					// 		roamingDrivers[i].Id,
					// 		roamingDrivers[i].Motivation,
					// 		roamingDrivers[i].Reputation,
					// 		roamingDrivers[i].Fatigue,
					// 		roamingDrivers[i].Regret,
					// 		roamingDrivers[i].GetRankingIndex())
					// }

					// COMMENT THIS SHIT IF SHIT GONE MAD
					//go dis.ComputeDriversRegret(roamingDrivers[:len(tasks)])

					// assigning task to driver
					for i := 0; i < len(tasks); i++ {

						fmt.Printf("[Dispatcher %d]Task %v with value of %v assigned to Driver %d with index of %v\n",
							e.Id, tasks[i].Id, tasks[i].FinalValue, roamingDrivers[i].Id, roamingDrivers[i].GetRankingIndex())
						roamingDrivers[i].Request <- Message{Task: tasks[i]}
					}

					// computing regret over here

					if len(roamingDrivers) > len(tasks) {
						noOfTasks := len(tasks)
						for k := 0; k < len(roamingDrivers[noOfTasks:]); k++ {
							fmt.Printf("[Dispatcher %d]Driver %d with index of %v not assigned to any tasks\n", e.Id, roamingDrivers[noOfTasks:][k].Id, roamingDrivers[noOfTasks:][k].GetRankingIndex())
							roamingDrivers[noOfTasks:][k].Request <- Message{Task: Task{Id: "-1"}} // send an invalid task.
						}
					}

				} else {
					fmt.Printf("[Dispatcher %d]No tasks available for assigning to the %d roaming drivers \n", e.Id, noOfRoamingDrivers)
					for k := 0; k < len(roamingDrivers); k++ {
						roamingDrivers[k].Request <- Message{Task: Task{Id: "-1"}} // send an invalid task.
					}
				}
			} else {
				fmt.Printf("[Dispatcher %d]No roaming drivers avialable for allocation.\n", e.Id)
			}
			fmt.Printf("[Dispatcher %d]===End of Assigning task(s) to roaming driver(s)===\n", e.Id)
		}
	}
}

type Dispatcher struct {
	E                    *Environment
	Response             chan DriverMatchingResult
	DriverAgents         map[int]*DriverAgent
	DriverAgentsResponse map[int]int // 0 - havent response, 1 - accepted, 2 - rejected
}

func SetupDispatcher(e *Environment) Dispatcher {
	return Dispatcher{
		E:                    e,
		Response:             make(chan DriverMatchingResult, 10000),
		DriverAgents:         make(map[int]*DriverAgent),
		DriverAgentsResponse: make(map[int]int),
	}
}

type DriverMatchingResult struct {
	Accept bool
	Id     int
}

// This function is called when dispatching - could be wrong.
func (dis *Dispatcher) ComputeDriversRegret(drivers []*DriverAgent) {
	// reintialize map
	dis.DriverAgents = make(map[int]*DriverAgent)
	dis.DriverAgentsResponse = make(map[int]int)

	// map array to map
	for k := 0; k < len(drivers); k++ {
		dis.DriverAgents[drivers[k].Id] = drivers[k]
		dis.DriverAgentsResponse[drivers[k].Id] = 0
	}

K:
	for {
		select {
		case r := <-dis.Response:
			fmt.Printf("[ComputeDriversRegret]Response: %v\n", r)
			if r.Accept {
				dis.DriverAgentsResponse[r.Id] = 1 // We know that the driver accepts the task
			} else {
				dis.DriverAgentsResponse[r.Id] = 2 // We know that the driver rejects the task
			}
			var driverCount = 0
			for _, v := range dis.DriverAgentsResponse {
				if v == 0 {
					break
				}
				driverCount++
				if driverCount == len(dis.DriverAgentsResponse) {
					break K
				}
			}
		}
	}

	for k, _ := range dis.DriverAgentsResponse {
		if dis.DriverAgentsResponse[k] == 1 {
			dis.DriverAgents[k].ComputeRegret()
		}
	}
	fmt.Printf("[ComputeDriversRegret]Finish computing regrets for drivers with tasks\n")
}

// Get k number of tasks ranking by the value.
func GetValuableTasks(k int, TaskQueue chan Task) []Task {
	tasks := make([]Task, 0)

K:
	// Grabs all tasks from the TaskQueue
	for {
		// Grab all tasks then break.
		if len(tasks) == 10 { // TODO: Set toggle max value when choosing list of valuable tasks
			break
		}
		select {
		case x := <-TaskQueue:
			//fmt.Printf("[GVT]Grab Task %d \n", x.Id)
			tasks = append(tasks, x)
		default:
			// if no more tasks in channel, break.
			break K
		}
	}

	// sort the tasks' value in descending order
	sort.SliceStable(tasks, func(i, j int) bool {
		return tasks[i].FinalValue > tasks[j].FinalValue
	})

	//fmt.Printf("[GVT]No of total tasks: %d, with no of roaming drivers: %d\n", len(tasks), k)
	if len(tasks) > k {
		for i := 0; i < len(tasks[k:]); i++ {
			//fmt.Printf("[GVT]Back to queue: Task %d with value of %d \n", tasks[k:][i].Id, tasks[k:][i].Value)
			TaskQueue <- tasks[k:][i]
		}
		return tasks[:k]
	}
	for i := 0; i < len(tasks); i++ {
		//fmt.Printf("[GVT] Return task %d for allocation \n", tasks[i].Id)
	}
	return tasks

}
