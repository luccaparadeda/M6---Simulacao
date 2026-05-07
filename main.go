package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"queuesim/config"
	"queuesim/queue"
	"queuesim/rng"
	"queuesim/sim"
)

func main() {
	configPath := flag.String("config", "simulation.yml", "path to YAML simulation config")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	configs := cfg.ToQueueConfigs()

	runs := make([]*sim.Simulator, 0, len(cfg.Simulation.Seeds))
	for i, seed := range cfg.Simulation.Seeds {
		fmt.Printf("Simulation: #%d\n", i+1)
		fmt.Printf("...simulating with random numbers (seed '%d')...\n", seed)

		r := rng.NewLCG(seed, cfg.Simulation.RNGPerSeed)
		s := sim.New(configs, r)
		s.ScheduleFirstArrival(cfg.Simulation.FirstArrival.Queue, cfg.Simulation.FirstArrival.Time)
		s.Run()
		runs = append(runs, s)
	}

	printFooter()
	printReport(configs, runs)
}

func printFooter() {
	fmt.Println("=========================================================")
	fmt.Println("=================    END OF SIMULATION   ================")
	fmt.Println("=========================================================")
	fmt.Println()
}

func printReport(configs []queue.Config, runs []*sim.Simulator) {
	fmt.Println("=========================================================")
	fmt.Println("======================    REPORT   ======================")
	fmt.Println("=========================================================")

	for _, c := range configs {
		fmt.Println("*********************************************************")
		fmt.Printf("Queue:   %s (%s)\n", c.ID, ggckLabel(c))
		if c.HasExternal {
			fmt.Printf("Arrival: %s ... %s\n", trimZero(c.ArrivalMin), trimZero(c.ArrivalMax))
		}
		fmt.Printf("Service: %s ... %s\n", trimZero(c.ServiceMin), trimZero(c.ServiceMax))
		fmt.Println("*********************************************************")
		fmt.Println("   State               Time               Probability")

		stateTimes := make(map[int]float64)
		losses := 0
		total := 0.0
		for _, s := range runs {
			q := s.Queues[c.ID]
			for k, v := range q.StateTimes {
				stateTimes[k] += v
				total += v
			}
			losses += q.Losses
		}
		states := make([]int, 0, len(stateTimes))
		for k := range stateTimes {
			states = append(states, k)
		}
		sort.Ints(states)
		for _, st := range states {
			t := stateTimes[st]
			p := 0.0
			if total > 0 {
				p = t / total * 100
			}
			fmt.Printf("%7d%21.4f%21.2f%%\n", st, t, p)
		}
		fmt.Println()
		fmt.Printf("Number of losses: %d\n\n", losses)
	}

	avg := 0.0
	for _, s := range runs {
		avg += s.Clock
	}
	avg /= float64(len(runs))
	fmt.Println("=========================================================")
	fmt.Printf("Simulation average time: %.4f\n", avg)
	fmt.Println("=========================================================")
}

func ggckLabel(c queue.Config) string {
	if c.Capacity < 0 {
		return fmt.Sprintf("G/G/%d", c.Servers)
	}
	return fmt.Sprintf("G/G/%d/%d", c.Servers, c.Capacity)
}

func trimZero(f float64) string {
	return fmt.Sprintf("%.1f", f)
}
