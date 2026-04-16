package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"queuesim/logger"
	"queuesim/queue"
	"queuesim/rng"
	"queuesim/sim"
)

func main() {
	seed := flag.Uint64("seed", 12345, "LCG seed for the random number generator")
	n := flag.Int("n", 100_000, "RNG budget (stop after consuming N random numbers)")
	logPath := flag.String("log", "", "path to write the event log (e.g. -log=simulation.csv)")
	jsonReport := flag.Bool("json", false, "output the final report as JSON instead of plain text")
	flag.Parse()

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
				{ID: "Q2", Probability: 1.0},
			},
		},
		{
			ID:         "Q2",
			Servers:    1,
			Capacity:   5,
			ServiceMin: 2,
			ServiceMax: 3,
		},
	}

	r := rng.NewLCG(*seed, *n)
	s := sim.New(configs, r)

	if *logPath != "" {
		queueIDs := make([]string, len(configs))
		for i, c := range configs {
			queueIDs[i] = c.ID
		}

		lg, err := newLogger(*logPath, queueIDs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating log file: %v\n", err)
			os.Exit(1)
		}
		defer lg.Close()
		s.Logger = lg
	}

	s.ScheduleFirstArrival("Q1", 1.5)
	s.Run()

	if *jsonReport {
		reportJSON(s)
	} else {
		s.Report()
	}

	if *logPath != "" {
		fmt.Fprintf(os.Stderr, "Log de eventos salvo em: %s\n", *logPath)
	}
}

// newLogger selects the log format strategy based on file extension.
// .json / .jsonl → JSONLogger, anything else → CSVLogger.
func newLogger(path string, queueIDs []string) (logger.Logger, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json", ".jsonl":
		return logger.NewJSON(path)
	default:
		return logger.NewCSV(path, queueIDs)
	}
}

type queueReport struct {
	ID          string             `json:"id"`
	Servers     int                `json:"servers"`
	Capacity    int                `json:"capacity"`
	Losses      int                `json:"losses"`
	States      []stateReport      `json:"states"`
}

type stateReport struct {
	State       int     `json:"state"`
	Time        float64 `json:"time"`
	Probability float64 `json:"probability"`
}

type fullReport struct {
	GlobalTime float64       `json:"global_time"`
	RNGCount   int           `json:"rng_count"`
	Queues     []queueReport `json:"queues"`
}

func reportJSON(s *sim.Simulator) {
	rep := fullReport{
		GlobalTime: s.Clock,
		RNGCount:   s.Rng.Count(),
	}
	for _, id := range s.Order {
		q := s.Queues[id]
		qr := queueReport{
			ID:       id,
			Servers:  q.Cfg.Servers,
			Capacity: q.Cfg.Capacity,
			Losses:   q.Losses,
		}
		total := 0.0
		states := make([]int, 0, len(q.StateTimes))
		for k := range q.StateTimes {
			states = append(states, k)
			total += q.StateTimes[k]
		}
		sort.Ints(states)
		for _, st := range states {
			t := q.StateTimes[st]
			p := 0.0
			if total > 0 {
				p = t / total
			}
			qr.States = append(qr.States, stateReport{State: st, Time: t, Probability: p})
		}
		rep.Queues = append(rep.Queues, qr)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(rep)
}
