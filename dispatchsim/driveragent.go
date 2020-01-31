package dispatchsim

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/paulmach/orb"
)

type DriverStatus int

const (
	Roaming DriverStatus = iota
	Matching
	Fetching
	Travelling
	Allocating
)

func (d DriverStatus) String() string {
	return [...]string{"Roaming", "Matching", "Fetching", "Travelling", "Allocating"}[d]
}

type DriverAgent struct {
	S                   *Simulation
	E                   *Environment
	Id                  int
	Name                string
	CurrentLocation     LatLng
	Waypoint            []LatLng
	NextLocation        LatLng
	DestinationLocation LatLng
	PendingTask         Task // store task during matching
	CurrentTask         Task // store task when fetching/travelling
	Status              DriverStatus
	Request             chan Message
	Recieve             chan Message // receive
	Send                chan string  // send
	TasksCompleted      int
	TaskHistory         []Task
	Motivation          float64
	Reputation          float64
	Fatigue             float64
	Regret              float64
	ChangeDestination   chan string
}

func CreateMultipleDrivers(startingDriverId, n int, e *Environment, s *Simulation) {
	for i := 0; i < n; i++ {
		driver := CreateDriver(startingDriverId, e, s)
		e.DriverAgents[startingDriverId] = &driver
		s.DriverAgents[startingDriverId] = &driver
		go e.DriverAgents[startingDriverId].ProcessTask2()
		startingDriverId++
	}
}

func CreateDriver(id int, e *Environment, s *Simulation) DriverAgent {
	return DriverAgent{
		S:                 s,
		E:                 e,
		Id:                id,
		Name:              "John",
		Status:            Roaming,
		Request:           make(chan Message),
		Recieve:           make(chan Message),
		Send:              make(chan string),
		TasksCompleted:    0,
		TaskHistory:       make([]Task, 0),
		Motivation:        100,
		Reputation:        0,
		Fatigue:           0,
		Regret:            0,
		ChangeDestination: make(chan string),
	}
}

func (d *DriverAgent) ProcessTask2() {
	go d.Drive()
	go d.RunComputeRegret()

	tick := time.Tick(d.E.S.MasterSpeed * time.Millisecond)
	for {
		select {
		case <-d.E.Quit:
			//fmt.Println("Driver " + strconv.Itoa(d.Id) + " ended his day")
			return
		case <-tick:

			switch d.Status {
			case Roaming:
				//fmt.Printf("[Driver %d - Env %d]%v \n", d.Id, d.E.Id, d.Status.String())
			case Allocating:
				//fmt.Printf("[Driver %d - Env %d]%v \n", d.Id, d.E.Id, d.Status.String())
				select {
				case t := <-d.Request:
					// fmt.Printf("Task address @ during matching: %p\n", &t)
					if t.Task.Id != "-1" {
						d.Status = Matching
						d.PendingTask = t.Task
						d.CurrentTask = t.Task
						//fmt.Printf("Driver %d got Task %v! \n", d.Id, t.Task.Id)
					} else {
						d.Status = Roaming
						//fmt.Printf("Driver %d did not get any tasks \n", d.Id)
					}
				default:

				}
			case Matching:
				if d.AcceptRide() == true {
					d.CurrentTask = d.PendingTask
					d.PendingTask = Task{Id: "null", FinalValue: 0.00} // bad way...
					d.Status = Fetching
					// uncomment the below one line code if shit gone mad
					d.E.S.Environments[d.CurrentTask.EnvironmentId].Dispatcher.Response <- DriverMatchingResult{Accept: true, Id: d.Id}
					d.E.S.Environments[d.CurrentTask.EnvironmentId].TaskMutex.Lock()
					d.E.S.Environments[d.CurrentTask.EnvironmentId].Tasks[d.CurrentTask.Id].WaitStart = d.E.S.SimulationTime
					d.E.S.Environments[d.CurrentTask.EnvironmentId].TaskMutex.Unlock()
					//d.ComputeRegret() IMPT! Uncomment if shit gone mad
					d.DestinationLocation = d.CurrentTask.StartCoordinate
					d.ChangeDestination <- "change message"
				} else {
					// send Tasks to TaskQueue
					d.E.TaskQueue <- d.CurrentTask
					d.PendingTask = Task{Id: "null", FinalValue: 0.00}
					d.Status = Roaming
					// set nil to d.Task
				}
				//fmt.Printf("[Driver %d - Env %d]%v \n", d.Id, d.E.Id, d.Status.String())
			case Fetching:
				//fmt.Printf("[Driver %d - Env %d]%v \n", d.Id, d.E.Id, d.Status.String())
				reached, ok := d.ReachTaskPosition()
				if ok == true && reached == true {
					//fmt.Printf("Driver %d has reached the passenger \n", d.Id)
					// driver has arrived at task location

					x := d.E.S.SimulationTime
					d.CurrentTask.WaitEnd = x
					d.E.S.Environments[d.CurrentTask.EnvironmentId].TaskMutex.Lock()
					d.E.S.Environments[d.CurrentTask.EnvironmentId].Tasks[d.CurrentTask.Id].WaitEnd = x
					d.E.S.Environments[d.CurrentTask.EnvironmentId].TaskMutex.Unlock()
					d.Status = Travelling
					d.DestinationLocation = d.CurrentTask.EndCoordinate
					v := [][]float64{}
					v = append(v, latLngToSlice(d.CurrentLocation))
					v = append(v, latLngToSlice(d.CurrentTask.EndCoordinate))
					c := SendFormat{2, 2, DriverInfoFormat{d.E.Id, d.Id, v}}
					d.E.S.Send <- structToString(c)
					//d.E.S.Send <- "2,2," + strconv.Itoa(d.E.Id) + "," + strconv.Itoa(d.Id) + "," + latLngToString(d.CurrentLocation) + "," + latLngToString(d.CurrentTask.EndPosition) // REMOVE
				}
			case Travelling:
				//fmt.Printf("[Driver %d - Env %d]%v \n", d.Id, d.E.Id, d.Status.String())
				reached, ok := d.ReachTaskDestination()
				if ok == true && reached == true {
					// driver has completed the task
					x := d.E.S.SimulationTime
					d.CurrentTask.TaskEnded = x
					//d.E.Tasks[d.CurrentTask.Id].TaskEnded = x
					d.E.S.Environments[d.CurrentTask.EnvironmentId].TaskMutex.Lock()
					d.E.S.Environments[d.CurrentTask.EnvironmentId].Tasks[d.CurrentTask.Id].TaskEnded = x
					d.E.S.Environments[d.CurrentTask.EnvironmentId].TaskMutex.Unlock()
					d.E.FinishQueue <- d.CurrentTask // put into FinishQueue
					d.CompleteTask()
					d.ComputeFatigue()
					d.ComputeMotivation()
					d.Status = Roaming
					// fmt.Printf("[Driver %d - Env %d]Completed task %v \n", d.Id, d.CurrentTask.Id, d.E.Id)
					// fmt.Printf("[Driver %d - Env %d]%v \n", d.Id, d.E.Id, d.Status.String())
					d.DriveToNextPoint()
				}
			}
			// migrate to another environment
			d.Migrate()
		}
	}
	// fmt.Printf("[Driver %d - Env %d]Driver agent function finished\n", d.Id, d.E.Id)
	log.Fatalf("[Driver %d] Ended", d.Id)
}

func (d *DriverAgent) Migrate() bool {
	// if the driver's location is in the not same
	cl := d.CurrentLocation
	point := orb.Point{cl.Lng, cl.Lat} //TODO pls fix the position of LatLng to LngLat
	inside := isPointInside(d.E.Polygon, point)
	if !inside {
		for _, v := range d.E.S.Environments {
			inside2 := isPointInside(v.Polygon, point)
			if inside2 {

				d.E.DriverAgentMutex.Lock()
				delete(d.E.DriverAgents, d.Id)
				d.E.DriverAgentMutex.Unlock()

				v.DriverAgentMutex.Lock()
				v.DriverAgents[d.Id] = d
				v.DriverAgentMutex.Unlock()
				d.E = v
				//fmt.Printf("[Driver %d]Migrating to %d\n", d.Id, v.Id)
				return true
			}
		}
	}
	return false
}
func (d *DriverAgent) Drive() {
	//1. Setting up start, destination and waypoint
	d.E.S.Send <- "2,1," + strconv.Itoa(d.E.Id) + "," + strconv.Itoa(d.Id) // REMOVE!
	c := SendFormat{2, 1, DriverInfoFormat{d.E.Id, d.Id, [][]float64{}}}
	d.E.S.Send <- structToString(c)
	var generatedNode bool = false
	for {
		select {
		case <-d.E.Quit:
			return
		case x := <-d.Recieve:
			switch x.CommandSecondType {
			case 0: // NO USE CASE FOR THIS YET
				//fmt.Printf("[Driver %d - Env %d]Saving random point (lat:%f,lng:%f) to current location \n", d.Id, x.LatLng.Lat, x.LatLng.Lng, d.E.Id)
			case 1:
				//fmt.Printf("[Driver %d - Env %d]Intialization of Driver\n", d.Id, d.E.Id)
				//1b. Recieve random current location, destination location and waypoint
				d.CurrentLocation = x.StartDestinationWaypoint.StartLocation
				d.DestinationLocation = x.StartDestinationWaypoint.DestinationLocation
				d.Waypoint = x.StartDestinationWaypoint.Waypoint[1:]

				if generatedNode == false {
					// 2. generate node
					v := [][]float64{}
					v = append(v, latLngToSlice(d.CurrentLocation))
					c := SendFormat{2, 3, DriverInfoFormat{d.E.Id, d.Id, v}}
					d.E.S.Send <- structToString(c)
					//d.E.S.Send <- "2,3," + strconv.Itoa(d.E.Id) + "," + strconv.Itoa(d.Id) + "," + latLngToString(d.CurrentLocation) // REMOVE
				}

			case 2:
				//fmt.Printf("[Driver %d - Env %d]Setting waypoint\n", d.Id, d.E.Id)
				// Waypoint must be more than 1. If is exactly 1, it means the driver and passenger is exactly on the same node
				if len(x.Waypoint) > 1 {
					d.NextLocation, d.Waypoint = x.Waypoint[1], x.Waypoint[1:]
					v := [][]float64{}
					v = append(v, latLngToSlice(d.CurrentLocation))
					v = append(v, latLngToSlice(d.NextLocation))
					c := SendFormat{2, 4, DriverInfoFormat{d.E.Id, d.Id, v}}
					d.E.S.Send <- structToString(c)
				}

			case 3:
				if x.Success == true {
					//fmt.Printf("[Driver %d - Env %d]Node generated\n", d.Id, d.E.Id)
					generatedNode = true
					d.DriveToNextPoint()
				}
			case 4:
				if x.Success == true {
					//fmt.Printf("[Driver %d - Env %d]Driver moved successfully\n", d.Id, d.E.Id)
					d.CurrentLocation = x.LocationArrived
					select {
					case <-d.ChangeDestination:
						v := [][]float64{}
						v = append(v, latLngToSlice(d.CurrentLocation))
						v = append(v, latLngToSlice(d.CurrentTask.StartCoordinate))
						c := SendFormat{2, 2, DriverInfoFormat{d.E.Id, d.Id, v}}
						d.E.S.Send <- structToString(c)

					default:
						d.DriveToNextPoint()
					}

				}
			case 5:
				//fmt.Printf("[Driver %d - Env %d]New random generated\n", d.Id, d.E.Id)
				d.CurrentLocation = x.StartDestinationWaypoint.StartLocation
				d.DestinationLocation = x.StartDestinationWaypoint.DestinationLocation
				d.Waypoint = x.StartDestinationWaypoint.Waypoint[1:]
				d.DriveToNextPoint()
			}
		}
	}
	fmt.Printf("[Driver %d - Env %d]Driver function finished\n", d.Id, d.E.Id)
}

func (d *DriverAgent) DriveToNextPoint() {
	// e.g [1,1],[2,2],[3,3],[4,4]
	switch noOfWaypoints := len(d.Waypoint); {
	case noOfWaypoints >= 2:
		//fmt.Printf("[Driver %d - Env %d]Driving to next point\n", d.Id, d.E.Id)
		d.NextLocation = d.Waypoint[0]
		v := [][]float64{}
		v = append(v, latLngToSlice(d.CurrentLocation))
		v = append(v, latLngToSlice(d.NextLocation))
		//c := SendFormat{2, 4, DriverInfoFormat{d.E.Id, d.Id, v}}
		//d.E.S.Send <- structToString(c)
		time.Sleep(200 * time.Millisecond)
		result := fmt.Sprintf("[2,4,%v,%v,[%v,%v]]", d.E.Id, d.Id, d.NextLocation.Lat, d.NextLocation.Lng)
		d.E.S.Recieve <- result
		d.Waypoint = d.Waypoint[1:]
	case noOfWaypoints == 1:
		//fmt.Printf("[Driver %d - Env %d]Driving to next point\n", d.Id, d.E.Id)
		d.NextLocation = d.Waypoint[0]
		v := [][]float64{}
		v = append(v, latLngToSlice(d.CurrentLocation))
		v = append(v, latLngToSlice(d.NextLocation))
		time.Sleep(200 * time.Millisecond)
		result := fmt.Sprintf("[2,4,%v,%v,[%v,%v]]", d.E.Id, d.Id, d.NextLocation.Lat, d.NextLocation.Lng)
		d.E.S.Recieve <- result
		// c := SendFormat{2, 4, DriverInfoFormat{d.E.Id, d.Id, v}}
		// d.E.S.Send <- structToString(c)
		d.Waypoint = d.Waypoint[:0]
	case noOfWaypoints == 0:
		//fmt.Printf("[Driver %d - Env %d]No more waypoint to drive\n", d.Id, d.E.Id)
		if d.Status == Roaming || d.Status == Allocating || d.Status == Matching {
			// Driver has reached the last point of the waypoint
			// Generate a new random destination and its waypoint
			v := [][]float64{}
			v = append(v, latLngToSlice(d.CurrentLocation))
			c := SendFormat{2, 5, DriverInfoFormat{d.E.Id, d.Id, v}}
			d.E.S.Send <- structToString(c)

		}
	}

}

// Accept rate
func (d *DriverAgent) AcceptRide() bool {
	return true
	// min := 0
	// max := 1
	// rand.Seed(time.Now().UnixNano())
	// n := min + rand.Intn(max-min+1)
	// if n == 0 {
	// 	d.CurrentTask.WaitStart = time.Now()
	// 	fmt.Println("Driver " + strconv.Itoa(d.Id) + " accepted ride")
	// 	return true
	// } else {
	// 	fmt.Println("Driver " + strconv.Itoa(d.Id) + " declined ride")
	// 	return false
	// }
}

func (d *DriverAgent) ReachTaskPosition() (bool, bool) {
	//fmt.Printf("[Driver %d - Env %d]Checking whether reach passenger \n", d.Id, d.E.Id)
	if d.CurrentLocation == d.CurrentTask.StartCoordinate {
		return true, true
	} else {
		return false, true
	}
}

func (d *DriverAgent) ReachTaskDestination() (bool, bool) {
	//fmt.Printf("[Driver %d - Env %d]Checking whether reach passenger's destination \n", d.Id, d.E.Id)
	if d.CurrentLocation == d.CurrentTask.EndCoordinate {
		return true, true
	} else {
		return false, true
	}
}

// After driver completed a task, update rating, taskscompleted, fatigue and motivation
func (d *DriverAgent) CompleteTask() (bool, bool) {

	getRating := d.CurrentTask.ComputeRating(d.S)
	fmt.Printf("Task %d with value of %d gave Driver %d a rating of %f \n", d.CurrentTask.Id, d.CurrentTask.Value, d.Id, getRating)

	var updatedRating float64 = 0
	if d.TasksCompleted > 0 {
		updatedRating = (d.Reputation + getRating) / 2
	} else {
		updatedRating = getRating
	}
	updatedTaskCompleted := d.TasksCompleted + 1

	d.TasksCompleted = updatedTaskCompleted
	d.Reputation = updatedRating
	d.TaskHistory = append(d.TaskHistory, d.CurrentTask)
	d.CurrentTask = Task{Id: "null", FinalValue: 0.00}

	return true, true
}

// use goroutines?
func (d *DriverAgent) ComputeFatigue() {
	d.Fatigue = d.Fatigue + 1
}

// use goroutines?
func (d *DriverAgent) ComputeMotivation() {
	d.Motivation = d.Motivation + 1
}

//

func (d *DriverAgent) RunComputeRegret() {

	x := time.Tick(2000 * time.Millisecond)

	for {
		select {
		case <-x:
			d.ComputeRegret()
		}
	}
}

// use goroutines? no...
func (d *DriverAgent) ComputeRegret() {
	//fmt.Printf("Computing Regret for Driver %d - Env %d\n", d.Id, d.E.Id)
	averageValueOfOrders := d.S.ComputeAverageValue(d)
	fmt.Printf("[Driver %d]Regret now: %f, Value of Order: %f, Average value of Orders: %f\n", d.Id, d.Regret, float64(d.CurrentTask.FinalValue), averageValueOfOrders)
	// Driver's current regret - current task's value + average value of orders which other drivers take at the point of time (with similar reputation)
	computedRegret := d.Regret - float64(d.CurrentTask.FinalValue) + averageValueOfOrders

	if computedRegret < 0 {
		d.Regret = 0
	} else {
		d.Regret = computedRegret
	}
	//fmt.Printf("[Driver %d]Regret Calculated: %f\n", d.Id, d.Regret)
}

func (d *DriverAgent) GetRankingIndex() float64 {
	fmt.Printf("[GetRankingIndex]Driver %d's stats - Motivation:%v, Reputation:%v, Fatigue:%v, Regret:%v\n",
		d.Id, d.Motivation, d.Reputation, d.Fatigue, d.Regret)
	return (d.Motivation * (d.Reputation - d.Fatigue)) + d.Regret
	// todo : norminlize reputation and fatigue
}
