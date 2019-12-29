package dispatchsim

import (
	"math/rand"
	"time"
)

type Task struct {
	Id            string
	StartPosition LatLng    // latlong
	EndPosition   LatLng    // latlong
	TaskCreated   time.Time // all time
	WaitStart     time.Time // all time
	WaitEnd       time.Time // all time
	TaskEnded     time.Time // all time
	Value         int
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

func CreateTaskFromOrder(o Order) Task {
	min := 1
	max := 10
	rand.Seed(time.Now().UnixNano())
	n := min + rand.Intn(max-min+1)

	return Task{
		Id:            o.Id,
		StartPosition: LatLng{Lat: o.PickUpLat, Lng: o.PickUpLng},
		EndPosition:   LatLng{Lat: o.DropOffLat, Lng: o.DropOffLng},
		TaskCreated:   time.Now(),
		Value:         n, // random value from 1 to 10 (for now... TODO!)
	}
}

// generate rating from 0 to 5
func (t *Task) ComputeRating() float64 {
	// gives rating
	return 5
}
