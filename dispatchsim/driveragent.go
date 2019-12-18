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
	E               *Environment
	Id              int
	Name            string
	CurrentLocation int  // change to lat,lng
	PendingTask     Task // store task during matching
	CurrentTask     Task // store task when fetching/travelling
	Status          DriverStatus
	Request         chan Message
	TasksCompleted  int
	TaskHistory     []Task
	Motivation      float64
	Reputation      float64
	Fatigue         float64
	Regret          float64
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
		TasksCompleted: 0,
		TaskHistory:    make([]Task, 0),
		Motivation:     100,
		Reputation:     0,
		Fatigue:        0,
		Regret:         0,
	}
}

func (d *DriverAgent) ProcessTask() {
	tick := time.Tick(d.E.MasterSpeed * time.Millisecond)
	for {
		select {
		case <-d.E.Quit:
			//fmt.Println("Driver " + strconv.Itoa(d.Id) + " ended his day")
			return
		case <-tick:
			switch d.Status {
			case Roaming:
				fmt.Println("Driver " + strconv.Itoa(d.Id) + " is " + d.Status.String())
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
