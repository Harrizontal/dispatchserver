package dispatchsim

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/paulmach/orb"
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

func latLngToSlice(ll LatLng) []float64 {
	x := []float64{}
	x = append(x, ll.Lat)
	x = append(x, ll.Lng)
	return x
}

// TODO: Fix the polygon in array
func twoLatLngtoArrayFloat(ll []LatLng) [][][]float64 {
	x := make([][][]float64, 0)
	z := make([][]float64, 0)
	for _, latLng := range ll {
		y := make([]float64, 0)
		y = append(y, latLng.Lat)
		y = append(y, latLng.Lng)
		z = append(z, y)
	}
	x = append(x, z)
	//fmt.Printf("[twoDimen]%v \n", x)
	return x
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

func structToString(st interface{}) string {
	out, err := json.Marshal(st)
	if err != nil {
		panic(err)
	}

	return string(out)
}

func ConvertToLatLng(in []interface{}) LatLng {
	if len(in) == 2 {
		lat := in[0].(float64)
		lng := in[1].(float64)
		return LatLng{Lat: lat, Lng: lng}
	} else {
		return LatLng{}
	}

}

func ConvertLngLatToLatLng(in []interface{}) LatLng {
	return LatLng{Lat: in[1].(float64), Lng: in[0].(float64)}
}

func ConvertToArrayLatLng(in []interface{}) []LatLng {
	var s []LatLng
	for _, value := range in {
		latlng := ConvertToLatLng(value.([]interface{}))
		s = append(s, latlng)
	}

	return s
}

func ConvertLatLngArrayToPolygon(ll []LatLng) orb.Polygon {
	poly := make(orb.Polygon, 0)
	ls := make(orb.LineString, 0)
	for _, v := range ll {
		ls = append(ls, orb.Point{v.Lat, v.Lng})
	}

	ring := orb.Ring(ls)
	poly = append(poly, ring)
	return poly
}
