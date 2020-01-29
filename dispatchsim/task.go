package dispatchsim

import (
	"math"
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
	// min := 1
	// max := 10
	// rand.Seed(time.Now().UnixNano())
	// n := min + rand.Intn(max-min+1)
	// fmt.Printf("o.Distance: %v\n", o.Distance)
	// fmt.Printf("S.RatePerKM: %v\n", e.S.RatePerKM)
	// fmt.Printf("o.Distance*e.S.RatePerKM: %v\n", o.Distance*e.S.RatePerKM)
	// fmt.Printf("value: %v\n", math.Round(float64(o.Distance*e.S.RatePerKM)*100)/100)

	return Task{
		Id:              o.Id,
		EnvironmentId:   e.Id,
		StartCoordinate: o.StartCoordinate,
		EndCoordinate:   o.EndCoordinate,
		PickUpLocation:  LatLng{Lat: o.PickUpLat, Lng: o.PickUpLng},
		DropOffLocation: LatLng{Lat: o.DropOffLat, Lng: o.DropOffLng},
		TaskCreated:     time.Now(),
		Value:           int(o.Distance), // random value from 1 to 10 (for now... TODO!)
		FinalValue:      math.Round(float64(o.Distance*e.S.RatePerKM)*100) / 100,
		Distance:        o.Distance,
	}
}

// TODO: generate rating from 0 to 5
func (t *Task) ComputeRating() float64 {
	// gives rating
	return 5
}
