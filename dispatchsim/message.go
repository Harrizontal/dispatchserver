package dispatchsim

type Message struct {
	CommandType              int
	CommandSecondType        int
	Task                     Task
	LatLng                   LatLng                   // case 0
	StartDestinationWaypoint StartDestinationWaypoint // case 1
	Waypoint                 []LatLng                 // case:2
	Success                  bool
	LocationArrived          LatLng
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

func ConvertToLatLng(in []interface{}) LatLng {
	lat := in[0].(float64)
	lng := in[1].(float64)
	return LatLng{Lat: lat, Lng: lng}
}

func ConvertToArrayLatLng(in []interface{}) []LatLng {
	var s []LatLng
	for _, value := range in {
		latlng := ConvertToLatLng(value.([]interface{}))
		s = append(s, latlng)
	}

	return s
}
