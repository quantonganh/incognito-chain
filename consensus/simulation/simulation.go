package main

type Simulation struct {
	scenario Scenario
	expected Expected
}

type Scenario struct {
	proposeComm map[int64]map[string][]int // timeSlot -> sender -> receiver
	voteComm    map[int64]map[string][]int
	sync        map[string][]string
}

type Expected struct {
}

var simulation *Simulation

func InitSimulation(committee []string, scenario []map[string]Scenario, expected []map[string]Expected) *Simulation {
	return simulation
}

func GetSimulation() *Simulation {
	if simulation == nil {
		simulation = new(Simulation)
		simulation.scenario = Scenario{
			proposeComm: make(map[int64]map[string][]int),
			voteComm:    make(map[int64]map[string][]int),
			sync:        make(map[string][]string),
		}
	}
	return simulation
}

func (s *Simulation) checkExpected() {

}
