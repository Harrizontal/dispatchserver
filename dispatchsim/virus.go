package dispatchsim

import (
	"math/rand"
	"time"
)

type VirusParameters struct {
	InitialInfectedDriversPercentage int       `json:"infected_drivers"`   // starts with mild first
	InfectedTaskPercentage           int       `json:"infected_tasks"`     // spread equaly to task. e.g 10% -> every 10 tasks, 5 tasks infected
	EvolveProbability                []float64 `json:"evolve_probability"` // 0 - 100.00
	SpreadProbability                []float64 `json:"spread_probability"`
	DriverMask                       int       `json:"driver_mask"`
	PassengerMask                    int       `json:"passenger_mask"`
	MaskEffectiveness                float64   `json:"mask_effectiveness"` //(0 - 100.00)
}

type Virus int

const (
	None Virus = 0 + iota
	Mild
	Moderate
	Severe
)

func (v Virus) String() string {
	return [...]string{"None", "Mild", "Moderate", "Severe"}[v]
}

func (v Virus) Evolve(p VirusParameters) Virus {

	a := int(Virus(v))

	if v == None {
		return v
	}

	//fmt.Printf("Current: %v\n", v)
	if a <= len(p.EvolveProbability) {
		rand.Seed(time.Now().UnixNano())
		r := rand.Float64()
		prob := p.EvolveProbability[v-1] / 100
		//fmt.Printf("%v <= %v", r, prob)
		if r <= prob {
			a++
			//fmt.Printf("Change to %v\n", Virus(a))
			return Virus(a)
		}
	}
	//fmt.Printf("Remain %v\n", Virus(a))
	return v
}

// Probability of the user spreading to other user
func SpreadVirus(p VirusParameters, wearMask bool) bool {
	if wearMask && p.MaskEffectiveness != 0 {
		r := rand.Float64()
		prob := p.MaskEffectiveness / 100
		//fmt.Printf("[SpreadVirus]%v > %v\n", r, prob)
		return r > prob
	} else {
		return true
	}
}

// Probability of the virus to air (cough, sneeze, etc)
func (v Virus) InAir(p VirusParameters) bool {
	a := int(Virus(v))
	if v == None {
		return false
	}

	r := rand.Float64()
	prob := p.SpreadProbability[a-1] / 100
	//fmt.Printf("[InAir]%v < %v\n", r, prob)
	//fmt.Printf("%v <= %v", r, prob)
	return r < prob
}

// Probability of the user recieving the virus
func RecieveVirus(p VirusParameters, wearMask bool) bool {
	if wearMask && p.MaskEffectiveness != 0 {
		r := rand.Float64()               // 0.05
		prob := p.MaskEffectiveness / 100 // 0.1
		//fmt.Printf("[RecieveVirus]%v > %v\n", r, prob)
		return r > prob
	} else {
		return true
	}
}
