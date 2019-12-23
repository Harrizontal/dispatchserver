package dispatchsim

import (
	"fmt"
	"strconv"
	"time"
)

type DriverStatus int

const (
	Roaming DriverStatus = iota
	Matching
	Fetching
	Travelling
)

func (d DriverStatus) String() string {
	return [...]string{"Roaming", "Matching", "Fetching", "Travelling"}[d]
}

type DriverAgent struct {
	E                   *Environment
	Id                  int
	Name                string
	CurrentLocation     LatLng
	Waypoint            []LatLng
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
}

// Random point
//TODO: Generate Address
func CreateDriver(id int, e *Environment) DriverAgent {
	return DriverAgent{
		E:              e,
		Id:             id,
		Name:           "John",
		Status:         Roaming,
		Request:        make(chan Message),
		Recieve:        make(chan Message),
		Send:           make(chan string),
		TasksCompleted: 0,
		TaskHistory:    make([]Task, 0),
		Motivation:     100,
		Reputation:     0,
		Fatigue:        0,
		Regret:         0,
	}
}

func (d *DriverAgent) ProcessTask() {
	tick := time.Tick(d.E.S.MasterSpeed * time.Millisecond)
	for {
		select {
		case <-d.E.Quit:
			//fmt.Println("Driver " + strconv.Itoa(d.Id) + " ended his day")
			return
		case <-tick:
			switch d.Status {
			case Roaming:
				fmt.Println("Driver " + strconv.Itoa(d.Id) + " is " + d.Status.String())
				d.E.S.Send <- "Driver " + strconv.Itoa(d.Id) + " is " + d.Status.String()
				select {
				case t := <-d.Request:
					d.Status = Matching
					d.PendingTask = t.Task
					d.CurrentTask = t.Task
					//print(&t)
					fmt.Println("Driver " + strconv.Itoa(d.Id) + " got Task! " + strconv.Itoa(t.Task.Id))
				default:

				}
			case Matching:
				fmt.Println("Driver " + strconv.Itoa(d.Id) + " is " + d.Status.String())

				if d.AcceptRide() == true {
					d.CurrentTask = d.PendingTask
					d.PendingTask = Task{Id: -1}
					d.Status = Fetching
					d.ComputeRegret()
				} else {
					// send Tasks to TaskQueue
					d.E.TaskQueue <- d.CurrentTask
					d.PendingTask = Task{Id: -1}
					d.Status = Roaming
					// set nil to d.Task
				}
				fmt.Println("Driver " + strconv.Itoa(d.Id) + " is " + d.Status.String())
			case Fetching:
				fmt.Println("Driver " + strconv.Itoa(d.Id) + " is " + d.Status.String())
			K:
				for {
					reached, ok := d.ReachTaskPosition()
					if ok == true {
						if reached == true {
							// driver has arrived at task location
							break K
						} else {
							fmt.Println("Driver " + strconv.Itoa(d.Id) + " is " + d.Status.String())
						}
					}
				}
				// driver has arrived at task location
				d.CurrentTask.WaitEnd = time.Now()
				d.Status = Travelling
			case Travelling:

			F:
				for {
					reached, ok := d.ReachTaskDestination()
					if ok == true {
						if reached == true {
							break F
						} else {
							fmt.Println("Driver " + strconv.Itoa(d.Id) + " is " + d.Status.String())
						}
					}
				}
				// driver has completed the task
				d.CurrentTask.TaskEnded = time.Now()
				d.E.FinishQueue <- d.CurrentTask // put into FinishQueue
				d.CompleteTask()
				d.ComputeFatigue()
				d.ComputeMotivation()
				d.Status = Roaming
				fmt.Println("Driver " + strconv.Itoa(d.Id) + " is completed the ride")
				fmt.Println("Driver " + strconv.Itoa(d.Id) + " is " + d.Status.String())
			}
		}
	}
}

func (d *DriverAgent) ProcessTask2() {
	go d.Drive()
	tick := time.Tick(d.E.S.MasterSpeed * time.Millisecond)
	for {
		select {
		case <-d.E.Quit:
			//fmt.Println("Driver " + strconv.Itoa(d.Id) + " ended his day")
			return
		case <-tick:
			switch d.Status {
			case Roaming:
			case Matching:
			case Fetching:
			case Travelling:

			}
		}
	}
}

func (d *DriverAgent) Drive() {
	//1. Request for a random point
	//stringArray := []string{"2", "1", strconv.Itoa(d.E.Id), strconv.Itoa(d.Id)}
	//justString := strings.Join(stringArray, ",")
	d.E.S.Send <- "2,4," + strconv.Itoa(d.E.Id) + "," + strconv.Itoa(d.Id) + ",[1,2],[3,4]"

	for {
		select {
		case <-d.E.Quit:
			return
		case x := <-d.Recieve:
			switch x.CommandSecondType {
			case 0:
				fmt.Printf("[Driver %d]Saving random point (lat:%f,lng:%f) to current location \n", d.Id, x.LatLng.Lat, x.LatLng.Lng)
				d.CurrentLocation = x.LatLng

			case 1:
				fmt.Printf("[Driver %d]Intialization of Driver (Start:%f, End:%f)\n", d.Id, x.StartDestinationWaypoint.StartLocation, x.StartDestinationWaypoint.DestinationLocation)
				d.CurrentLocation = x.StartDestinationWaypoint.DestinationLocation
				d.DestinationLocation = x.StartDestinationWaypoint.DestinationLocation
				d.Waypoint = x.StartDestinationWaypoint.Waypoint

				fmt.Printf("Current Location: %v, DestinationLocation: %v, Waypoint: %v \n", d.CurrentLocation, d.DestinationLocation, d.Waypoint)
			case 2:
				fmt.Printf("[Driver %d]Setting waypoint\n", d.Id)
				d.Waypoint = x.Waypoint
			case 3:
				if x.Success == true {
					fmt.Printf("[Driver %d] Node generated\n", d.Id)
				} else {
					fmt.Printf("[Driver %d] Node failed to generate\n", d.Id)
				}
			case 4:
				if x.Success == true {
					fmt.Printf("[Driver %d] Driver moved successfully\n", d.Id)
				} else {
					fmt.Printf("[Driver %d] Node failed to move\n", d.Id)
				}
			}

			// or remove everything below, and just go to the next location in the array
			switch d.Status {
			case Roaming: // or is matching too...
				// go next location in the array (send)
				// check got task
			case Fetching:
				// go next location in the array (send)
			case Travelling:
				// go next location in the array (send)
			}
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

// TODO: implement movement
func (d *DriverAgent) ReachTaskPosition() (bool, bool) {
	return true, true
}

// TODO: implement movement
func (d *DriverAgent) ReachTaskDestination() (bool, bool) {
	return true, true
}

// After driver completed a task, update rating, taskscompleted, fatigue and motivation
func (d *DriverAgent) CompleteTask() (bool, bool) {

	getRating := d.CurrentTask.ComputeRating()
	fmt.Printf("Task %d with value of %d gave Driver %d a rating of %f \n", d.CurrentTask.Id, d.CurrentTask.Value, d.Id, getRating)

	var updatedRating float64 = 0
	if d.TasksCompleted > 0 {
		updatedRating = (d.Reputation + getRating) / 2
	} else {
		updatedRating = d.CurrentTask.ComputeRating()
	}
	updatedTaskCompleted := d.TasksCompleted + 1

	d.TasksCompleted = updatedTaskCompleted
	d.Reputation = updatedRating
	d.TaskHistory = append(d.TaskHistory, d.CurrentTask)
	d.CurrentTask = Task{Id: -1}

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

// use goroutines?
func (d *DriverAgent) ComputeRegret() {
	fmt.Printf("Computing Regret for Driver %d\n", d.Id)
	fmt.Printf("Regret now: %f, Value of Order: %f, Average value of Orders: %f\n", d.Regret, float64(d.CurrentTask.Value), d.E.ComputeAverageValue(d.Reputation))
	computedRegret := d.Regret - float64(d.CurrentTask.Value) + d.E.ComputeAverageValue(d.Reputation)

	if computedRegret < 0 {
		d.Regret = 0
	} else {
		d.Regret = computedRegret
	}
	fmt.Printf("Computing Regret for Driver %d. Calculated: %f\n", d.Id, d.Regret)
}

func (d *DriverAgent) GetRankingIndex() float64 {
	return (d.Motivation * (d.Reputation - d.Fatigue)) + d.Regret
}
