package dispatchsim

import (
	"math"
	"math/rand"
	"time"
)

type Task struct {
	Id              string
	EnvironmentId   int
	StartCoordinate LatLng    // Lat lng from dataset
	EndCoordinate   LatLng    // Lat lng from dataset
	PickUpLocation  LatLng    // latlong - start of waypoint
	DropOffLocation LatLng    // latlong - end of waypoint
	TaskCreated     time.Time // all time
	WaitStart       time.Time // all time
	WaitEnd         time.Time // all time
	TaskEnded       time.Time // all time
	Value           int
	FinalValue      float64
	Distance        float64
	Valid           bool // valid -> task is reachable from a driver. invalid -> task is not reachable from 1 or more driver
}

// generate random start position, end position
// func CreateTask(id int) Task {
// 	min := 1
// 	max := 10
// 	rand.Seed(time.Now().UnixNano())
// 	n := min + rand.Intn(max-min+1)

// 	return Task{
// 		Id:          id,
// 		TaskCreated: time.Now(),
// 		Value:       n, // random value from 1 to 10 (for now... TODO!)
// 	}
// }

func CreateTaskFromOrder(o Order, e *Environment) Task {

	var taskValue float64 = 0.00
	// calculate taskvalue
	switch e.S.TaskParameters.TaskValueType {
	case "random":
		taskValue = float64(GenerateRandomValue(1, 10)) * e.S.TaskParameters.ValuePerKM
	case "distance":
		taskValue = math.Round(float64(o.Distance*e.S.TaskParameters.ValuePerKM)*100) / 100
	}

	task := Task{
		Id:              o.Id,
		EnvironmentId:   e.Id,
		StartCoordinate: o.StartCoordinate,
		EndCoordinate:   o.EndCoordinate,
		PickUpLocation:  LatLng{Lat: o.PickUpLat, Lng: o.PickUpLng},
		DropOffLocation: LatLng{Lat: o.DropOffLat, Lng: o.DropOffLng},
		TaskCreated:     time.Now(),
		Value:           int(o.Distance), // random value from 1 to 10 (for now... TODO!)
		FinalValue:      taskValue,
		Distance:        o.Distance,
		Valid:           true,
	}

	return task
}

func GenerateRandomValue(min int, max int) int {
	rand.Seed(time.Now().UnixNano())
	n := min + rand.Intn(max-min+1)
	return n
}

// Give rating to driver after completed the task
func (t *Task) ComputeRating(s *Simulation) float64 {
	switch s.TaskParameters.ReputationGivenType {
	case "random":
		return float64(GenerateRandomValue(0, 5))
	case "fixed":
		return s.TaskParameters.ReputationValue
	default:
		return float64(GenerateRandomValue(0, 5)) // return int
	}
}
