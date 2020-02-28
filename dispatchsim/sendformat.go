package dispatchsim

type SendFormat struct {
	Command       int         `json:"command_first"`
	CommandSecond int         `json:"command_second"`
	Data          interface{} `json:"data"`
}

type DriverInfoFormat struct {
	EnvironmentId int         `json:"environment_id"`
	DriverId      int         `json:"driver_id"`
	LatLngs       [][]float64 `json:"lat_lngs"`
}

type OrderInfoFormat struct {
	OrderId         int       `json:"order_id"`
	PickUpLocation  []float64 `json:"pick_up_coordinates"`  // lat lng...
	DropOffLocation []float64 `json:"drop_off_coordinates"` // lat lng
}

type StatsDriverStatusFormat struct {
	Time              string `json:"time"`
	RoamingDrivers    int    `json:"no_of_roaming_drivers"`
	FetchingDrivers   int    `json:"no_of_picking_up_drivers"`
	TravellingDrivers int    `json:"no_of_travelling_drivers"`
}

type StatsRegretFormat struct {
	Time          string               `json:"time"`
	DriversRegret []DriverRegretFormat `json:"drivers_regret"`
}

type StatsVirusFormat struct {
	Time           string `json:"time"`
	TasksToDrivers int    `json:"tasks_to_drivers"`
	DriversToTasks int    `json:"drivers_to_tasks"`
}

type StatsEarningFormat struct {
	Time    string  `json:"time"`
	Max     float64 `json:"max_earning"`
	Average float64 `json:"average_earning"`
	Min     float64 `json:"min_earning"`
}

type StatsCurrentEarningFormat struct {
	Time    string  `json:"time"`
	Max     float64 `json:"current_max_earning"`
	Average float64 `json:"current_average_earning"`
	Min     float64 `json:"current_min_earning"`
}

type DriverRegretFormat struct {
	EnvironmentId int     `json:"environment_id"` // not need for displaying in the front end
	DriverId      int     `json:"driver_id"`
	Regret        float64 `json:"regret"`
}
