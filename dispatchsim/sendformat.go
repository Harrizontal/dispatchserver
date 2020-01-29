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

type StatsFormat struct {
	Time              string `json:"time"`
	RoamingDrivers    int    `json:"no_of_roaming_drivers"`
	FetchingDrivers   int    `json:"no_of_picking_up_drivers"`
	TravellingDrivers int    `json:"no_of_travelling_drivers"`
}
