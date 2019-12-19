package dispatchsim

import (
	"encoding/json"
	"log"
)

func stringToArrayInt(s string) []int {
	var ints []int
	err := json.Unmarshal([]byte(s), &ints)
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Printf("%v", ints)
	return ints
}
