package dispatchsim

import (
	"encoding/json"
	"fmt"
	"log"
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
