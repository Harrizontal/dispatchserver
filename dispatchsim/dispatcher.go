package dispatchsim

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// Structure of the Dispatcher
type Dispatcher struct {
	E                    *Environment
	Response             chan DriverMatchingResult
	DriverAgents         map[int]*DriverAgent
	DriverAgentsResponse map[int]int // 0 - havent response, 1 - accepted, 2 - rejected
	MatchingDrivers      chan DriverAgent
	RejectedDrivers      chan DriverAgent
	MatchingTasks        chan Task
	NoOfTaskTaken        int
	DispatchCount        int
}

type DriverMatchingResult struct {
	Accept bool
	Id     int
}

// Intialize the Dispatcher
func SetupDispatcher(e *Environment) Dispatcher {
	return Dispatcher{
		E:                    e,
		Response:             make(chan DriverMatchingResult, 10000),
		DriverAgents:         make(map[int]*DriverAgent),
		DriverAgentsResponse: make(map[int]int),
		MatchingDrivers:      make(chan DriverAgent, 1000),
		RejectedDrivers:      make(chan DriverAgent, 1000),
		MatchingTasks:        make(chan Task, 1000),
		NoOfTaskTaken:        0,
		DispatchCount:        0, // for count the number of times the dispatcher execute the algorithm
	}
}

// Dispatcher's algorithm
func (dis *Dispatcher) dispatcher3(e *Environment) {
	fmt.Printf("[Dispatcher %d]Awaiting to start\n", e.Id)
	<-dis.E.S.StartDispatchers
	fmt.Printf("[Dispatcher %d]Started\n", e.Id)
	ticker := time.Tick(time.Duration(dis.E.S.DispatcherParameters.DispatchInterval) * time.Millisecond) // default is 500
	for {
	K:
		select {
		case <-e.S.Stop: // main stop
			fmt.Printf("[Dispatcher %d]Stop by main\n", e.Id)
			return
		case <-e.Stop: // stop by environment
			fmt.Printf("[Dispatcher %d]Stop\n", e.Id)
			return
		case <-ticker: // For every tick, it executes the algorithm
			fmt.Printf("[>Dispatcher]Started\n")
			simTime := dis.E.S.SimulationTime

			// Intialize arrays to store drivers and its variables such as fatigue, reputation, etc
			roamingDrivers := make([]*DriverAgent, 0)
			mapDrivers := make(map[int]int)
			storeRoamingDriverId := make([]int, 0)
			storeMotivation := make([]float64, 0)
			storeFatigue := make([]float64, 0)
			storeReputation := make([]float64, 0)
			storeRegret := make([]float64, 0)

			// For all the drivers in the boundary (or environment - e),  store the motivation, fatigue, reputation and regret
			count := 0
			for _, v := range e.DriverAgents {
				storeMotivation = append(storeMotivation, v.Motivation)
				storeFatigue = append(storeFatigue, v.Fatigue)
				storeReputation = append(storeReputation, v.Reputation)
				storeRegret = append(storeRegret, v.Regret)
				mapDrivers[v.Id] = count
				if v.Status == Roaming && v.Valid {
					// If the driver has a status of roaming and its valid (valid - meaning the driver is ABLE to fetch passengers), store in the array
					// Some of the drivers may stuck in an island (driver happened to spawn in isolated road networks)
					storeRoamingDriverId = append(storeRoamingDriverId, v.Id)
					v.Status = Allocating // change the driver to allocating - to prevent double allocation if there are more than one boundary set by the user
				}
				count++
			}

			// Calculate the Min and Max for motivation, reputation, fatgiue and regret
			motivationMinMax := CalculateMinMax(storeMotivation)
			reputationMinMax := CalculateMinMax(storeReputation)
			fatigueMinMax := CalculateMinMax(storeFatigue)
			regretMinMax := CalculateMinMax(storeRegret)

			// For each roaming drivers, we calculate the ranking index with respect to Min and Max of the motivation, reputation, fatigue and regret
			for _, d := range storeRoamingDriverId {
				driverAgent := e.DriverAgents[d]
				driverAgent.RankingIndex = driverAgent.GetRankingIndexParams(
					motivationMinMax,
					reputationMinMax,
					fatigueMinMax,
					regretMinMax,
					storeMotivation[mapDrivers[d]],
					storeReputation[mapDrivers[d]],
					storeFatigue[mapDrivers[d]],
					storeRegret[mapDrivers[d]])
				roamingDrivers = append(roamingDrivers, driverAgent)
			}

			noOfRoamingDrivers := len(roamingDrivers)

			// If there are no roaming drivers available in the boundary, lets exit the dispatcher algorithm and wait for the next tick
			if noOfRoamingDrivers == 0 {
				break K
			}

			// If there are roaming drivers, grab a list of tasks based on the number of roaming drivers in the boundary
			tasks := dis.GetValuableTasks2(e.TaskQueue, noOfRoamingDrivers)

			// If there are no tasks available in the boundary
			if len(tasks) == 0 {
				// change all drivers' status from allocating to roaming
				for _, d := range roamingDrivers {
					d.Status = Roaming
				}
				// Exit the dispatcher algorithm and wait for the next tick
				break K
			}

			// sort drivers according to ranking index
			sort.SliceStable(roamingDrivers, func(i, j int) bool {
				return roamingDrivers[i].RankingIndex > roamingDrivers[j].RankingIndex
			})

			for _, d := range roamingDrivers {
				Log.Printf("%v [Dispatcher %v]Driver %d with ranking index %v and total earning %v\n", simTime, dis.E.Id, d.Id, d.RankingIndex, d.TotalEarnings)
			}

			noOfTasks := len(tasks)
			// If there are more tasks than the roaming drivers, push back the rest of tasks to the TaskQueue
			if noOfTasks > noOfRoamingDrivers {

				// cut tasks
				extraTasks := tasks[noOfRoamingDrivers:]
				tasks = tasks[:noOfRoamingDrivers]
				go func() {
					// we need to push away the task back to queue (goroutine)
					for i := 0; i < len(extraTasks); i++ {
						e.TaskQueue <- extraTasks[i]
						dis.NoOfTaskTaken--
					}
				}()
			} else if noOfTasks < noOfRoamingDrivers {
				// If there are more roaming drivers than tasks, those drivers at the back of array (with the lowest ranking index)
				// status will be converted to allocating to Roaming

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

			// Now we got and equal amount of drivers and tasks to match
			noOfRoamingDrivers = len(roamingDrivers)
			noOfTasks = len(tasks)

			// If is not equal, the algorithm of the dispatcher is broken. Need to revisit on how the way is coded
			// Note: So far so good!
			if noOfTasks != noOfRoamingDrivers {
				panic("NOT EQUAL!")
			}

			// Let's begin the actual dispatcher of tasks to drivers
			dis.DispatchCount++
			for d := 0; d < noOfRoamingDrivers; d++ {
				// Get the distance and waypoints (array of Latlng) of the driver's current location to the start location of the task
				_, _, distance, waypoints := dis.E.S.RN.GetWaypoint(roamingDrivers[d].CurrentLocation, tasks[d].StartCoordinate)

				// Get the waypoints of the start location of task to the end location of task
				_, _, _, waypoints2 := dis.E.S.RN.GetWaypoint(tasks[d].StartCoordinate, tasks[d].EndCoordinate)

				// If the distance return is valid (not infinite), let's assign the task to driver
				if distance != math.Inf(1) {
					fmt.Printf("[Dispatcher]Task %v with value %v, distance %v to Driver %d with ranking index of %v\n", tasks[d].Id, tasks[d].FinalValue, tasks[d].Distance, roamingDrivers[d].Id, roamingDrivers[d].RankingIndex)
					Log.Printf("%v [Dispatcher %d, Count: %d]Task %v (%v) with value %v, distance %v to Driver %d\n", simTime, dis.E.Id, dis.DispatchCount, tasks[d].Id, tasks[d].TaskCreated, tasks[d].FinalValue, tasks[d].Distance, roamingDrivers[d].Id)
					sdw := &StartDestinationWaypoint{
						StartLocation:       roamingDrivers[d].CurrentLocation,
						DestinationLocation: tasks[d].StartCoordinate,
						Waypoint:            waypoints,
					}
					sdw2 := &StartDestinationWaypoint{
						Waypoint: waypoints2,
					}
					roamingDrivers[d].Request <- Message{Task: tasks[d], StartDestinationWaypoint: *sdw, StartDestinationWaypoint2: *sdw2}
					//fmt.Printf("[Dispatcher](Done)Task %v to Driver %d\n", tasks[d].Id, roamingDrivers[d].Id)
				} else {
					// If the distance give infinite (it means drivers cannot reach the task)
					// We deemed the driver as Invalid (valid = false), and the dispatcher will not account this driver into the algorithm
					roamingDrivers[d].Valid = false // turn valid to false for driver, - this driver is in an island
					e.TaskQueue <- tasks[d]         // We push back the task back to the queue since the driver cannot take it
					dis.NoOfTaskTaken--

					// There are ways to improve this algorithm when the drivers, but I believe this is the most safest and less computative.

				}
			}
			fmt.Printf("[<Dispatcher]Ended\n")
		}
	}
	// The dispatcher should not reach to this point of the code. If so, let's crash the simulation.
	panic("unreachable")
}

// This function is not used
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

// This function will get a list of tasks ranking from the highest to lowest in terms of the value
// This function accepts a limit in how much tasks to get from the boundary
func (dis *Dispatcher) GetValuableTasks2(TaskQueue chan Task, limit int) []Task {
	//fmt.Printf("[GetValuableTasks2]Getting tasks (Limit:%v)\n", limit)
	tasks := make([]Task, 0)

K:
	// Grabs all tasks from the TaskQueue
	for {
		// Grab all tasks then break.
		if len(tasks) == limit {
			break K
		}
		select {
		case x := <-TaskQueue:
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

	return tasks

}
