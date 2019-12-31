package dispatchsim

type SendFormat struct {
	Command       int         `json:"command_first`
	CommandSecond int         `json:"commmand_second"`
	Data          interface{} `json:"data"`
}

type DriverInfoFormat struct {
	EnvironmentId int `json:"environment_id"`
	DriverId      int `json:"driver_id"`
	LatLngs       [][]float64
}
