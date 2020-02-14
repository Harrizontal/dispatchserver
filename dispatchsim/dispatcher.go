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
	MatchingLimit        int
	MatchingDrivers      chan DriverAgent
	RejectedDrivers      chan DriverAgent
	MatchingTasks        chan Task
	NoOfTaskTaken        int
}

func SetupDispatcher(e *Environment) Dispatcher {
	return Dispatcher{
		E:                    e,
		Response:             make(chan DriverMatchingResult, 10000),
		DriverAgents:         make(map[int]*DriverAgent),
		DriverAgentsResponse: make(map[int]int),
		MatchingLimit:        500,
		MatchingDrivers:      make(chan DriverAgent, 1000),
		RejectedDrivers:      make(chan DriverAgent, 1000),
		MatchingTasks:        make(chan Task, 1000),
		NoOfTaskTaken:        0,
	}
}

type DriverMatchingResult struct {
	Accept bool
	Id     int
}

func (dis *Dispatcher) dispatcher3(e *Environment) {
	fmt.Printf("[Dispatcher %d]Awaiting to start\n", e.Id)
	<-dis.E.S.StartDispatchers
	fmt.Printf("[Dispatcher %d]Started\n", e.Id)
	ticker := time.Tick(500 * time.Millisecond)
	for {
	K:
		select {
		case <-ticker:
			//fmt.Printf("[>Dispatcher]Started %v\n", dis.NoOfTaskTaken)

			// get tasks from environment
			tasks := dis.GetValuableTasks2(e.TaskQueue, dis.MatchingLimit)

			if len(tasks) == 0 {
				break K
			}

			roamingDrivers := make([]*DriverAgent, 0)

			// get all roaming drivers
			e.DriverAgentMutex.Lock()
			mmRepFat := e.S.GetMinMaxReputationFatigue()
			for _, v := range e.DriverAgents {
				if v.Status == Roaming && v.Valid {
					roamingDrivers = append(roamingDrivers, v)
					v.Status = Allocating // change roaming to allocating to prevent double task allocation when migrating
				}
			}
			e.DriverAgentMutex.Unlock()

			noOfRoamingDrivers := len(roamingDrivers)

			// sort drivers according to ranking index
			sort.SliceStable(roamingDrivers, func(i, j int) bool {
				return roamingDrivers[i].GetRankingIndex(&mmRepFat) > roamingDrivers[j].GetRankingIndex(&mmRepFat)
			})

			noOfTasks := len(tasks)

			//fmt.Printf("[Dispatcher]Intial - Drivers:%v, Tasks:%v\n", noOfRoamingDrivers, noOfTasks)
			if noOfTasks > noOfRoamingDrivers {

				// cut tasks
				extraTasks := tasks[noOfRoamingDrivers:]
				tasks = tasks[:noOfRoamingDrivers]
				//fmt.Printf("[Dispatcher]tasks>drivers - Drivers:%v, Tasks:%v\n", len(noOfRoamingDrivers), len(tasks))
				go func() {
					// we need to push away the task back to queue (goroutine)
					for i := 0; i < len(extraTasks); i++ {
						e.TaskQueue <- extraTasks[i]
						dis.NoOfTaskTaken--
					}
				}()
			} else if noOfTasks < noOfRoamingDrivers {

				// cut drivers
				extraDrivers := roamingDrivers[noOfTasks:]
				roamingDrivers = roamingDrivers[:noOfTasks]

				go func() {
					// we need to push the drivers to roaming
					for i := 0; i < len(extraDrivers); i++ {
						extraDrivers[i].Status = Roaming
					}
				}()
			}

			noOfRoamingDrivers = len(roamingDrivers)
			noOfTasks = len(tasks)
			//fmt.Printf("[Dispatcher]Left - Drivers:%v, Tasks:%v\n", noOfRoamingDrivers, noOfTasks)

			if noOfTasks != noOfRoamingDrivers {
				panic("NOT EQUAL!")
			}

			go func() {
				for d := 0; d < noOfRoamingDrivers; d++ {
					_, _, distance, _ := dis.E.S.RN.G_GetWaypoint(roamingDrivers[d].CurrentLocation, tasks[d].StartCoordinate)
					if distance != math.Inf(1) {
						//fmt.Printf("[Dispatcher]Task %v to Driver %d with status of %v\n", tasks[d].Id, roamingDrivers[d].Id, roamingDrivers[d].Status)
						roamingDrivers[d].Request <- Message{Task: tasks[d]}
						//fmt.Printf("[Dispatcher](Done)Task %v to Driver %d\n", tasks[d].Id, roamingDrivers[d].Id)
					} else {
						//fmt.Printf("[Dispatcher]Task %v to Driver %d (rejected)\n", tasks[d].Id, roamingDrivers[d].Id)
						roamingDrivers[d].Valid = false // turn valid to false for driver, - this driver is in an island
						e.TaskQueue <- tasks[d]
						dis.NoOfTaskTaken--
						//fmt.Printf("[Dispatcher](done)Task %v to Driver %d (rejected)\n", tasks[d].Id, roamingDrivers[d].Id)
					}
				}
				fmt.Printf("[<Dispatcher]Ended\n")
			}()

		}
	}

	panic("unreachable")
}

// This dispatcher caters to island drivers
func (dis *Dispatcher) dispatcher2(e *Environment) {
	ticker := time.Tick(500 * time.Millisecond)
	for {
		//K:
		select {
		case <-ticker:
			// if dis.E.S.OM.IsDistributing {
			// 	break K
			// }
			fmt.Printf("[>Dispatcher]Started %v\n", dis.NoOfTaskTaken)
			noOfRoamingDrivers := 0
			roamingDrivers := make([]*DriverAgent, 0)

			// get all roaming drivers
			e.DriverAgentMutex.Lock()
			mmRepFat := e.S.GetMinMaxReputationFatigue()
			for _, v := range e.DriverAgents {
				if v.Status == Roaming {
					roamingDrivers = append(roamingDrivers, v)
					noOfRoamingDrivers++
					v.Status = Allocating // change roaming to allocating to prevent double task allocation when migrating
				}
			}
			e.DriverAgentMutex.Unlock()

			// sort drivers according to ranking index
			sort.SliceStable(roamingDrivers, func(i, j int) bool {
				return roamingDrivers[i].GetRankingIndex(&mmRepFat) > roamingDrivers[j].GetRankingIndex(&mmRepFat)
			})
			// get sorted tasks according to final value
			tasks := dis.GetValuableTasks(noOfRoamingDrivers, e.TaskQueue, dis.MatchingLimit)

			// if drivers over the limit, we put the rest of the drivers back to roaming.
			if len(tasks) < dis.MatchingLimit {
				overLimitDrivers := roamingDrivers[len(tasks):]
				roamingDrivers = roamingDrivers[:len(tasks)]
				//fmt.Printf("[Dispatcher]Total Overlimit Roaming Drivers: %v\n", len(overLimitDrivers))
				for i := 0; i < len(overLimitDrivers); i++ {
					overLimitDrivers[i].Status = Roaming
				}
			} else {
				if len(roamingDrivers) > dis.MatchingLimit {
					overLimitDrivers := roamingDrivers[dis.MatchingLimit:]
					roamingDrivers = roamingDrivers[:dis.MatchingLimit]
					//fmt.Printf("[Dispatcher]Total Overlimit Roaming Drivers: %v\n", len(overLimitDrivers))
					for i := 0; i < len(overLimitDrivers); i++ {
						overLimitDrivers[i].Status = Roaming
					}
				}
			}

			var approvedRD = make([]*DriverAgent, 0)
			var approvedRDIndex = make([]int, 0)
			var matchedT = make([]*Task, 0)
			var unmatchedT = make([]*Task, 0)

			d := 0
			t := 0
			//ad := 0
			aad := 0
			invalidCount := 0
			enableTrack := true
			noOfDrivers := len(roamingDrivers) // minus 1 to match array
			noOfTasks := len(tasks)
			//fmt.Printf("[Dispatcher]Info -> Drivers:%v Tasks:%v\n", noOfDrivers, noOfTasks)
		F:
			for d < noOfDrivers {
				//fmt.Printf("d:%d\n", d)
				_, found := Find(approvedRDIndex, d)
				if !found {
				L:
					for t < noOfTasks {
						//fmt.Printf("d:%d t:%d\n", d, t)
						_, found2 := Find(approvedRDIndex, d)
						if found2 {
							d++
							break L
						}
						_, _, distance, _ := dis.E.S.RN.G_GetWaypoint(roamingDrivers[d].CurrentLocation, tasks[t].StartCoordinate)
						if distance != math.Inf(1) {
							invalidCount = 0
							//fmt.Printf("Driver %d is approved to Task %v\n", roamingDrivers[d].Id, tasks[t].Id)
							approvedRD = append(approvedRD, roamingDrivers[d])
							approvedRDIndex = append(approvedRDIndex, d)
							matchedT = append(matchedT, &tasks[t])
							d++
							t++
							if enableTrack == false {
								d = aad
								enableTrack = true
								break L
							}

						} else {
							//fmt.Printf("Driver %d cant handle Task %v due to task's location\n", roamingDrivers[d].Id, tasks[t].Id)
							if enableTrack {
								enableTrack = false
								aad = d
							}
							invalidCount++
							// (d + 1) == (noOfDrivers) - intial implementation
							if invalidCount == (noOfDrivers - len(approvedRD)) {
								// this task cannot be done
								//fmt.Printf("Task %v cannot be matched\n", tasks[t].Id)
								unmatchedT = append(unmatchedT, &tasks[t]) // order is lost. Todo: fix
								d = aad                                    // reset to the driver that is being match
								t++                                        // proceed to the next task, since this task cannot be match
								enableTrack = true                         // renable so the next unmatched task
								break F
							}
							d++
						}
					}
					if noOfTasks == 0 {
						// there is no tasks available, lets break out of the main for loop
						break F
					}
				} else {
					//fmt.Printf("Checking on next driver\n")
					d++
				}
			}

			for i := 0; i < noOfDrivers; i++ {
				_, found := Find(approvedRDIndex, i)
				if !found {
					fmt.Printf("Driver %d is leftover\n", roamingDrivers[i].Id)
					roamingDrivers[i].Request <- Message{Task: Task{Id: "null"}}
				}
			}

			//TODO: check for chance level.
			for i := 0; i < noOfTasks; i++ {
				_, found := FindTask(matchedT, tasks[i].Id)
				if !found {
					fmt.Printf("Task %v is leftover\n", tasks[i].Id)
					// e.TaskQueue <- tasks[i]
					dis.E.TaskMutex.Lock()
					dis.E.Tasks[tasks[i].Id].Valid = false
					dis.E.TaskMutex.Unlock()
				}
			}

			for i := 0; i < len(approvedRD); i++ {
				approvedRD[i].Request <- Message{Task: *matchedT[i]}
			}

			// for i := 0; i < len(unmatchedT); i++ {
			// 	//fmt.Printf("[Dispatcher]Task %v turns to invalid. Will not display no map anymore.\n", unmatchedT[i])
			// 	dis.E.TaskMutex.Lock()
			// 	dis.E.Tasks[unmatchedT[i].Id].Valid = false
			// 	dis.E.TaskMutex.Unlock()
			// 	//TODO: put a chance attribute. When up to like 3 chance, this task is deemed invalid. 1 chance = 1 chance of getting dispatch to driver
			// }

			fmt.Printf("[<Dispatcher]Ended\n")

		}
	}
}

func Find(slice []int, val int) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

func FindTask(slice []*Task, val string) (int, bool) {
	for i, item := range slice {
		if item.Id == val {
			return i, true
		}
	}
	return -1, false
}

// Matching algorithm here
func (dis *Dispatcher) dispatcher(e *Environment) {
	ticker := time.Tick(500 * time.Millisecond)
	for {
	K:
		select {
		case <-e.Quit:
			return
		case <-ticker:
			fmt.Printf("[>Dispatcher %d - %v]Start dispatch algorithm\n", e.Id, dis.E.S.SimulationTime)
			if dis.E.S.OM.IsDistributing {
				fmt.Printf("[?Dispatcher %d - %v]Exit dispatch algorithm\n", e.Id, dis.E.S.SimulationTime)
				break K
			}
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

			// sort drivers according to ranking index
			sort.SliceStable(roamingDrivers, func(i, j int) bool {
				return roamingDrivers[i].GetRankingIndex(&mmRepFat) > roamingDrivers[j].GetRankingIndex(&mmRepFat)
			})

			fmt.Printf("[Dispatcher]Total Roaming Drivers: %v\n", len(roamingDrivers))
			if len(roamingDrivers) > dis.MatchingLimit {
				overLimitDrivers := roamingDrivers[dis.MatchingLimit:]
				roamingDrivers = roamingDrivers[:dis.MatchingLimit]
				fmt.Printf("[Dispatcher]Total Overlimit Roaming Drivers: %v\n", len(overLimitDrivers))
				for i := 0; i < len(overLimitDrivers); i++ {
					overLimitDrivers[i].Status = Roaming
				}
			}

			fmt.Printf("[Dispatcher]Total Limit Roaming Drivers: %v\n", len(roamingDrivers))

			//fmt.Printf("NoOfRoamingDrivers: %d\n", noOfRoamingDrivers)

			if noOfRoamingDrivers > 0 {
				//fmt.Println("[Dispatcher]Start allocation")
				tasks := dis.GetValuableTasks(noOfRoamingDrivers, e.TaskQueue, dis.MatchingLimit)
				if len(tasks) > 0 {
					fmt.Printf("[Dispatcher %d]Assigning task(s) to roaming driver(s)\n", e.Id)

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

					// for i := 0; i < len(approvedRD); i++ {
					// 	fmt.Printf("[approvedRD]Driver %v\n", approvedRD[i].Id)
					// }

					// for i := 0; i < len(rejectedRD); i++ {
					// 	fmt.Printf("[rejectedRD]Driver %v\n", rejectedRD[i].Id)
					// }

					// for i := 0; i < len(leftoverRD); i++ {
					// 	fmt.Printf("[leftoverRD]Driver %v\n", leftoverRD[i].Id)
					// }

					// for i := 0; i < len(unmatchedT); i++ {
					// 	fmt.Printf("[unmatchedT]Task %v - Returning to queue\n", unmatchedT[i].Id)
					// 	//e.TaskQueue <- *unmatchedT[i]
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

					fmt.Printf("[<Dispatcher]End of dispatch\n")

				} else {
					fmt.Printf("[<Dispatcher %d]No tasks available for assigning to the %d roaming drivers \n", e.Id, noOfRoamingDrivers)
					for k := 0; k < len(roamingDrivers); k++ {
						roamingDrivers[k].Request <- Message{Task: Task{Id: "null"}} // send an invalid task.
					}
				}
			} else {
				fmt.Printf("[<Dispatcher %d]No roaming drivers avialable for allocation.\n", e.Id)
			}
			fmt.Printf("[<Dispatcher %d]End of Assigning task(s) to roaming driver(s)\n", e.Id)
		}
	}
	panic("Dispatcher crashed")
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
func (dis *Dispatcher) GetValuableTasks(k int, TaskQueue chan Task, limit int) []Task {
	fmt.Printf("[GetValuableTasks] Getting tasks (%v,%v)\n", k, limit)
	tasks := make([]Task, 0)

K:
	// Grabs all tasks from the TaskQueue
	for {
		// Grab all tasks then break.
		if len(tasks) == limit || len(tasks) == k { // TODO: Set toggle max value when choosing list of valuable tasks
			break
		}
		select {
		case x := <-TaskQueue:
			//fmt.Printf("[GVT]Grab Task %d \n", x.Id)
			tasks = append(tasks, x)
			dis.NoOfTaskTaken++
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
	// if there is lesser roaming drivers than total no of tasks gotten from queue
	if len(tasks) > k {
		for i := 0; i < len(tasks[k:]); i++ {
			//fmt.Printf("[GVT]Back to queue: Task %d with value of %d \n", tasks[k:][i].Id, tasks[k:][i].Value)
			TaskQueue <- tasks[k:][i]
			dis.NoOfTaskTaken--
		}
		fmt.Printf("[GetValuableTasks]Finish getting tasks\n")
		return tasks[:k]
	}

	fmt.Printf("[GetValuableTasks]Finish getting tasks\n")
	return tasks

}

func (dis *Dispatcher) GetValuableTasks2(TaskQueue chan Task, limit int) []Task {
	//fmt.Printf("[GetValuableTasks2]Getting tasks (Limit:%v)\n", limit)
	tasks := make([]Task, 0)

K:
	// Grabs all tasks from the TaskQueue
	for {
		// Grab all tasks then break.
		if len(tasks) == limit { // TODO: Set toggle max value when choosing list of valuable tasks
			break K
		}
		select {
		case x := <-TaskQueue:
			//fmt.Printf("[GetValuableTasks2]Get Task %v from queue\n", x.Id)
			tasks = append(tasks, x)
			dis.NoOfTaskTaken++
		default:
			// if no more tasks in channel, break.
			break K
		}
	}

	// sort the tasks' value in descending order
	sort.SliceStable(tasks, func(i, j int) bool {
		return tasks[i].FinalValue > tasks[j].FinalValue
	})

	fmt.Printf("[GetValuableTasks2]Finish getting tasks - %v\n", len(tasks))
	return tasks

}
