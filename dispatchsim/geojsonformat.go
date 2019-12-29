package dispatchsim

import "time"

type GeoJSONFormat struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
}

type Feature struct {
	Type       string     `json:"type"`
	Geometry   Geometry   `json:"geometry"`
	Properties Properties `json:"properties"`
}

type Geometry struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

type Properties struct {
	Type        string      `json:"type"`
	Information interface{} `json:"information"` // accepts DriverFormat or Task Format
}

type DriverFormat struct {
	Id     int          `json:"id"`
	Status DriverStatus `json:"status"`
}

type TaskFormat struct {
	Id            string    `json:"id"`
	StartPosition []float64 `json:"start_coordinates"`
	EndPosition   []float64 `json:"end_coordinates"`
	WaitStart     time.Time `json:"start_wait"`
	WaitEnd       time.Time `json:"end_wait"`
	TaskEnd       time.Time `json:"task_end"`
}
