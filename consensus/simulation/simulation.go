package main

type Simulation struct {
	scenario Scenario
	expected Expected
}

type Scenario struct {
	proposeComm *[]int
	voteComm    *[]int
	commitComm  *[]int
	sync        *map[string][]string
}

type Expected struct {
}

var simulation *Simulation

func InitSimulation(committee []string, scenario []map[string]Scenario, expected []map[string]Expected) *Simulation {
	return simulation
}

func GetSimulation() *Simulation {
	return simulation
}

func (s *Simulation) checkExpected() {

}
