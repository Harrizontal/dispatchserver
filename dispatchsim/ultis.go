package dispatchsim

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

func stringToArrayInt(s string) []int {
	var ints []int
	err := json.Unmarshal([]byte(s), &ints)
	if err != nil {
		fmt.Printf("Cannot convert stringToArrayString\n")
		log.Fatal(err)
	}
	//fmt.Printf("%v", ints)
	return ints
}

func stringToArrayString(s string) interface{} {
	var v interface{}
	err := json.Unmarshal([]byte(s), &v)
	if err != nil {
		fmt.Printf("Cannot convert stringToArrayString %v\n", s)
		log.Fatal(err)
	}
	//fmt.Printf("%v", ints)
	return v
}

func latLngToString(ll LatLng) string {
	var str string = "["
	var s []string
	s = append(s, fmt.Sprint(ll.Lat))
	s = append(s, fmt.Sprint(ll.Lng))
	sLatLng := strings.Join(s, ",")
	str = str + sLatLng + "]"
	return str
}
func latlngToArrayFloat(ll LatLng) []float64 {
	var s []float64
	s = append(s, ll.Lng)
	s = append(s, ll.Lat)
	return s
}

func arrayLatLngToString(ll []LatLng) {
	var str string = "["
	for k, latLng := range ll {
		var str2 string = "["
		var s []string
		s = append(s, fmt.Sprint(latLng.Lat))
		s = append(s, fmt.Sprint(latLng.Lng))
		sLatLng := strings.Join(s, ",")
		// fmt.Println(asd)
		str2 = str2 + sLatLng + "]"
		if k != len(ll)-1 {
			str2 = str2 + ","
		}

		str = str + str2
	}
	str = str + "]"
	fmt.Println(str)
}
