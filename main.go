package main

import (
	"queuesim/queue"
	"queuesim/rng"
	"queuesim/sim"
)

func main() {
	// Cenário de validação: Tandem G/G/2/3 -> G/G/1/5
	configs := []queue.Config{
		{
			ID:          "Q1",
			Servers:     2,
			Capacity:    3,
			ArrivalMin:  1,
			ArrivalMax:  4,
			ServiceMin:  3,
			ServiceMax:  4,
			HasExternal: true,
			Routes: []queue.Route{
				{ID: "Q2", Probability: 1.0}, // 100% para Q2
			},
		},
		{
			ID:         "Q2",
			Servers:    1,
			Capacity:   5,
			ServiceMin: 2,
			ServiceMax: 3,
			// sem rotas => saída do sistema
		},
	}

	r := rng.NewLCG(12345, 100_000)
	s := sim.New(configs, r)
	s.ScheduleFirstArrival("Q1", 1.5)
	s.Run()
	s.Report()
}
