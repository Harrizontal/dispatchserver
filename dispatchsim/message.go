package dispatchsim

type Message struct {
	CommandType               int
	CommandSecondType         int
	Task                      Task
	LatLng                    LatLng                   // case 0
	StartDestinationWaypoint  StartDestinationWaypoint // case 1
	StartDestinationWaypoint2 StartDestinationWaypoint // for task - start to end
	Waypoint                  []LatLng                 // case:2
	Success                   bool
	LocationArrived           LatLng
}

//1
type StartDestinationWaypoint struct {
	StartLocation       LatLng
	DestinationLocation LatLng
	Waypoint            []LatLng
}

type LatLng struct {
	Lat float64
	Lng float64
}

type RecieveFormat struct {
	Command       int
	CommandSecond int
	Data          interface{}
}
