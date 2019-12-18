package dispatchsim

import (
	"fmt"
	"sort"
	"time"
)

// Implement algorithm here
func dispatcher(e *Environment) {
	fmt.Println("Start Dispatcher")
	tick := time.Tick(e.MasterSpeed * time.Millisecond) // TODO: master timer
	for {
		select {
		case <-e.Quit:
			//fmt.Println("Close Dispatcher")
			return
		case <-tick:

			// rank all drivers by Driver Ranking Index (DRI) descending order
			sort.SliceStable(e.DriverAgents, func(i, j int) bool {
				return e.DriverAgents[i].GetRankingIndex() > e.DriverAgents[j].GetRankingIndex()
			})

			noOfRoamingDrivers := 0
			var roamingDrivers = make([]*DriverAgent, 0)
			for i := 0; i < len(e.DriverAgents); i++ {
				if e.DriverAgents[i].Status == Roaming {
					//fmt.Printf("Adding Roaming Driver %d %p \n", e.DriverAgents[i].Id, &e.DriverAgents[i])
					roamingDrivers = append(roamingDrivers, &e.DriverAgents[i])
					noOfRoamingDrivers++
				}
			}

			if noOfRoamingDrivers > 0 {
				fmt.Println("[Dispatcher] Start allocation")
				tasks := GetValuableTasks(noOfRoamingDrivers, e.TaskQueue)
				// rank passengers
				// rank orders
				if len(tasks) > 0 {
					sort.SliceStable(tasks, func(i, j int) bool {
						return tasks[i].Value > tasks[j].Value
					})

					for i := 0; i < len(tasks); i++ {
						fmt.Printf("[Dispatcher] Task %d with value of %d is to be allocated \n", tasks[i].Id, tasks[i].Value)
					}

					sort.SliceStable(roamingDrivers, func(i, j int) bool {
						return roamingDrivers[i].GetRankingIndex() > roamingDrivers[j].GetRankingIndex()
					})

					for i := 0; i < len(roamingDrivers); i++ {
						fmt.Printf("[Dispatcher] Roaming Driver %d with movtivation %f, reputation %f , fatigue %f and regret %f. Ranking Index: %f\n",
							roamingDrivers[i].Id,
							roamingDrivers[i].Motivation,
							roamingDrivers[i].Reputation,
							roamingDrivers[i].Fatigue,
							roamingDrivers[i].Regret,
							roamingDrivers[i].GetRankingIndex())
					}

					for i := 0; i < len(e.DriverAgents); i++ {
						fmt.Printf("[Dispatcher] Driver %d with movtivation %f, reputation %f , fatigue %f and regret %f. Ranking Index: %f\n",
							e.DriverAgents[i].Id,
							e.DriverAgents[i].Motivation,
							e.DriverAgents[i].Reputation,
							e.DriverAgents[i].Fatigue,
							e.DriverAgents[i].Regret,
							e.DriverAgents[i].GetRankingIndex())
					}

					// assign first task to first driver
					for i := 0; i < len(tasks); i++ {
						roamingDrivers[i].Request <- Message{Task: tasks[i]}
					}

				} else {
					fmt.Printf("[Dispatcher] No more tasks available for assigning to %d roaming drivers \n", noOfRoamingDrivers)
				}

			} else {
				fmt.Println("[Dispatcher] No roaming dirvers for allocation.")
			}

		default:

		}
	}

}

func GetValuableTasks(k int, TaskQueue chan Task) []Task {
	tasks := make([]Task, 0)

K:
	for {
		if len(tasks) == 100 { // TODO: Set toggle max value when choosing list of valuable tasks
			break
		}
		select {
		case x := <-TaskQueue:
			fmt.Printf("[GVT] Grab Task %d \n", x.Id)
			tasks = append(tasks, x)
		default:
			// if no more tasks in channel, break.
			break K
		}
	}
	fmt.Printf("[GVT] No of total tasks: %d, with no of roaming drivers: %d\n", len(tasks), k)
	if len(tasks) > k {
		for i := 0; i < len(tasks[k:]); i++ {
			fmt.Printf("[GVT] Back to queue: Task %d \n", tasks[k:][i].Id)
			TaskQueue <- tasks[k:][i]
		}
		return tasks[:k]
	}
	for i := 0; i < len(tasks); i++ {
		fmt.Printf("[GVT] Return task %d for allocation \n", tasks[i].Id)
	}
	return tasks

}
