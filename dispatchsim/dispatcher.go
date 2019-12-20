package dispatchsim

import (
	"fmt"
	"sort"
	"time"
)

// Implement algorithm here
func dispatcher(e *Environment) {
	fmt.Println("[Dispatcher]Start Dispatcher")
	tick := time.Tick(e.S.MasterSpeed * time.Millisecond) // TODO: master timer

	for {
		select {
		case <-e.Quit:
			return
		case <-tick:

			noOfRoamingDrivers := 0
			var roamingDrivers = make([]*DriverAgent, 0)
			for _, v := range e.DriverAgents2 {
				if v.Status == Roaming {
					roamingDrivers = append(roamingDrivers, v)
					noOfRoamingDrivers++
					// change roaming to allocating status!! todo!! - for migrating drivers
				}
			}

			sort.SliceStable(roamingDrivers, func(i, j int) bool {
				return roamingDrivers[i].GetRankingIndex() > roamingDrivers[j].GetRankingIndex()
			})

			fmt.Printf("NoOfRoamingDrivers: %d\n", noOfRoamingDrivers)

			if noOfRoamingDrivers > 0 {
				fmt.Println("[Dispatcher]Start allocation")
				tasks := GetValuableTasks(noOfRoamingDrivers, e.TaskQueue)
				if len(tasks) > 0 {
					for i := 0; i < len(tasks); i++ {
						fmt.Printf("[Dispatcher]Task %d with value of %d is to be allocated \n", tasks[i].Id, tasks[i].Value)
					}

					// Sort drivers' Driver Ranking Index in descending order!
					sort.SliceStable(roamingDrivers, func(i, j int) bool {
						return roamingDrivers[i].GetRankingIndex() > roamingDrivers[j].GetRankingIndex()
					})

					// for printing - can delete!
					for i := 0; i < len(roamingDrivers); i++ {
						fmt.Printf("[Dispatcher]Roaming Driver %d with movtivation %f, reputation %f , fatigue %f and regret %f. Ranking Index: %f\n",
							roamingDrivers[i].Id,
							roamingDrivers[i].Motivation,
							roamingDrivers[i].Reputation,
							roamingDrivers[i].Fatigue,
							roamingDrivers[i].Regret,
							roamingDrivers[i].GetRankingIndex())
					}

					// assigning task to driver
					for i := 0; i < len(tasks); i++ {
						roamingDrivers[i].Request <- Message{Task: tasks[i]}
					}
				} else {
					fmt.Printf("[Dispatcher]No more tasks available for assigning to the %d roaming drivers \n", noOfRoamingDrivers)
				}
			} else {
				fmt.Println("[Dispatcher]No roaming dirvers for allocation.")
			}

		}
	}
}
func GetValuableTasks(k int, TaskQueue chan Task) []Task {
	tasks := make([]Task, 0)

K:
	// Grabs all tasks from the TaskQueue!
	for {
		if len(tasks) == 100 { // TODO: Set toggle max value when choosing list of valuable tasks
			break
		}
		select {
		case x := <-TaskQueue:
			fmt.Printf("[GVT]Grab Task %d \n", x.Id)
			tasks = append(tasks, x)
		default:
			// if no more tasks in channel, break.
			break K
		}
	}

	// sort the tasks' value in descending order
	sort.SliceStable(tasks, func(i, j int) bool {
		return tasks[i].Value > tasks[j].Value
	})

	fmt.Printf("[GVT]No of total tasks: %d, with no of roaming drivers: %d\n", len(tasks), k)
	if len(tasks) > k {
		for i := 0; i < len(tasks[k:]); i++ {
			fmt.Printf("[GVT]Back to queue: Task %d with value of %d \n", tasks[k:][i].Id, tasks[k:][i].Value)
			TaskQueue <- tasks[k:][i]
		}
		return tasks[:k]
	}
	for i := 0; i < len(tasks); i++ {
		fmt.Printf("[GVT] Return task %d for allocation \n", tasks[i].Id)
	}
	return tasks

}
