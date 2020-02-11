package dispatchsim

import (
	"fmt"
	"math"
	"sort"
	"time"
)

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

// Matching algorithm here
func (dis *Dispatcher) dispatcher(e *Environment) {
	//tick := time.Tick(time.Duration(e.S.DispatcherParameters.DispatchInterval) * time.Millisecond) // TODO: master timer
	ticker := time.Tick(500 * time.Millisecond)
	for {
		select {
		case <-e.Quit:
			return
		case <-ticker:
			fmt.Printf("[Dispatcher %d - %v]Start dispatch algorithm\n", e.Id, dis.E.S.SimulationTime)
			noOfRoamingDrivers := 0
			var roamingDrivers = make([]*DriverAgent, 0)

			e.DriverAgentMutex.Lock()
			mmRepFat := e.S.GetMinMaxReputationFatigue()
			// fmt.Printf("minmax: %v", mmRepFat)
			for _, v := range e.DriverAgents {
				//fmt.Printf("[Dispatcher %d] Driver %d -> %v\n", e.Id, v.Id, v.Status)
				if v.Status == Roaming {
					roamingDrivers = append(roamingDrivers, v)
					noOfRoamingDrivers++
					v.Status = Allocating // change roaming to allocating to prevent double task allocation when migrating
				}
			}
			e.DriverAgentMutex.Unlock()
			//TODO: Duplicate code?

			//fmt.Printf("NoOfRoamingDrivers: %d\n", noOfRoamingDrivers)

			if noOfRoamingDrivers > 0 {
				//fmt.Println("[Dispatcher]Start allocation")
				tasks := GetValuableTasks(noOfRoamingDrivers, e.TaskQueue)
				if len(tasks) > 0 {
					fmt.Printf("[Dispatcher %d]Assigning task(s) to roaming driver(s)\n", e.Id)

					// sort drivers according to ranking index
					sort.SliceStable(roamingDrivers, func(i, j int) bool {
						return roamingDrivers[i].GetRankingIndex(&mmRepFat) > roamingDrivers[j].GetRankingIndex(&mmRepFat)
					})

					var approvedRD = make([]*DriverAgent, 0)
					var matchedT = make([]*Task, 0)
					var rejectedRD = make([]*DriverAgent, 0)
					var unmatchedT = make([]*Task, 0)
					var leftoverRD = make([]*DriverAgent, 0)

					d := 0
					t := 0
					ad := 0
					noOfDrivers := len(roamingDrivers) // minus 1 to match array
					noOfTasks := len(tasks)
					fmt.Printf("[Dispatcher Info] %v %v\n", noOfDrivers, noOfTasks)
					for d < noOfDrivers {
						//fmt.Printf("a\n")
					L:
						for t < noOfTasks {
							_, _, distance, _ := dis.E.S.RN.G_GetWaypoint(roamingDrivers[d].CurrentLocation, tasks[t].StartCoordinate)
							if distance != math.Inf(1) {
								fmt.Printf("Driver %d is approved to Task %v\n", roamingDrivers[d].Id, tasks[t].Id)
								approvedRD = append(approvedRD, roamingDrivers[d])
								matchedT = append(matchedT, &tasks[t])
								d++
								t++
								ad++
								break L
							} else {
								fmt.Printf("Driver %d is cant handle Task %v due to task's location\n", roamingDrivers[d].Id, tasks[t].Id)
								//rejectedRD = append(rejectedRD, roamingDrivers[d])
								d++
								break L
							}
							//fmt.Printf("b\n")
						}

						if d == noOfDrivers && t < noOfTasks {
							unmatchedT = append(unmatchedT, &tasks[t])
							d = ad
							t++
						}

						if t == noOfTasks && d < noOfDrivers {
							fmt.Printf("Driver %d is leftover\n", roamingDrivers[d].Id)
							leftoverRD = append(leftoverRD, roamingDrivers[d])
							d++
						}

						//fmt.Printf("c\n")
					}

					for i := 0; i < len(approvedRD); i++ {
						fmt.Printf("[approvedRD]Driver %v\n", approvedRD[i].Id)
					}

					for i := 0; i < len(rejectedRD); i++ {
						fmt.Printf("[rejectedRD]Driver %v\n", rejectedRD[i].Id)
					}

					for i := 0; i < len(leftoverRD); i++ {
						fmt.Printf("[leftoverRD]Driver %v\n", leftoverRD[i].Id)
					}

					for i := 0; i < len(unmatchedT); i++ {
						fmt.Printf("[unmatchedT]Task %v - Returning to queue\n", unmatchedT[i].Id)
						//e.TaskQueue <- *unmatchedT[i]
					}

					// fmt.Printf("[TaskLeftOver info] %v %v\n", t, noOfTasks)
					// if t < noOfTasks {
					// 	for i := 0; i < len(tasks[t:]); i++ {
					// 		fmt.Printf("[TaskLeftOver]Task %v\n", tasks[t:][i].Id)
					// 		e.TaskQueue <- tasks[t:][i]
					// 	}
					// }

					// assigned task to drivers who have actual access to the task
					for i := 0; i < len(approvedRD); i++ {
						approvedRD[i].Request <- Message{Task: *matchedT[i]}
					}

					for i := 0; i < len(rejectedRD); i++ {
						rejectedRD[i].Request <- Message{Task: Task{Id: "null"}}
					}

					for i := 0; i < len(leftoverRD); i++ {
						leftoverRD[i].Request <- Message{Task: Task{Id: "null"}}
					}

					fmt.Printf("[Dispatcher]End of dispatch\n")
					// // once we ensure that driver is able to go to task (driver might be in a deadlock location), we put the driver in approvedRoamingDrivers
					// // else, goes to rejectedRoamingDrivers
					// var approvedRoamingDrivers = make([]*DriverAgent, 0) // those drivers that CAN drive to their order's start coordinate
					// var rejectedRoamingDrivers = make([]*DriverAgent, 0) // those drivers that CANNOT drive to their order's start coordinate

					// // assigning task to driver
					// for i := 0; i < len(tasks); i++ {
					// 	_, _, distance, _ := dis.E.S.RN.G_GetWaypoint(roamingDrivers[i].CurrentLocation, tasks[i].StartCoordinate)
					// 	if distance != math.Inf(1) {
					// 		approvedRoamingDrivers = append(approvedRoamingDrivers, roamingDrivers[i])
					// 	} else {
					// 		rejectedRoamingDrivers = append(rejectedRoamingDrivers, roamingDrivers[i])
					// 	}
					// }

					// // send invalid task to rejected roaming drivers
					// for i := 0; i < len(rejectedRoamingDrivers); i++ {
					// 	rejectedRoamingDrivers[i].Request <- Message{Task: Task{Id: "null"}} // send an invalid task
					// }

					// // assigning task to driver
					// for i := 0; i < len(tasks); i++ {
					// 	fmt.Printf("[Dispatcher %d]Task %v with value of %v assigned to Driver %d\n",
					// 		e.Id, tasks[i].Id, tasks[i].FinalValue, approvedRoamingDrivers[i].Id)
					// 	approvedRoamingDrivers[i].Request <- Message{Task: tasks[i]}
					// }

					// // if there is more roaming drivers than the tasks
					// if len(approvedRoamingDrivers) > len(tasks) {
					// 	noOfTasks := len(tasks)
					// 	for k := 0; k < len(approvedRoamingDrivers[noOfTasks:]); k++ {
					// 		//fmt.Printf("[Dispatcher %d]Driver %d with index of %v not assigned to any tasks\n", e.Id, roamingDrivers[noOfTasks:][k].Id, roamingDrivers[noOfTasks:][k].GetRankingIndex())
					// 		fmt.Printf("[Dispatcher] Driver %d going back to Roaming\n", approvedRoamingDrivers[k].Id)
					// 		approvedRoamingDrivers[noOfTasks:][k].Request <- Message{Task: Task{Id: "null"}} // send an invalid task.
					// 	}
					// }

				} else {
					fmt.Printf("[Dispatcher %d]No tasks available for assigning to the %d roaming drivers \n", e.Id, noOfRoamingDrivers)
					for k := 0; k < len(roamingDrivers); k++ {
						roamingDrivers[k].Request <- Message{Task: Task{Id: "null"}} // send an invalid task.
					}
				}
			} else {
				fmt.Printf("[Dispatcher %d]No roaming drivers avialable for allocation.\n", e.Id)
			}
			fmt.Printf("[Dispatcher %d]End of Assigning task(s) to roaming driver(s)\n", e.Id)
		}
	}
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
