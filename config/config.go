package config

import (
	"fmt"
	"os"

	"queuesim/queue"

	"gopkg.in/yaml.v3"
)

type Range struct {
	Min float64 `yaml:"min"`
	Max float64 `yaml:"max"`
}

type Route struct {
	To          string  `yaml:"to"`
	Probability float64 `yaml:"probability"`
}

type Queue struct {
	ID       string  `yaml:"id"`
	Servers  int     `yaml:"servers"`
	Capacity *int    `yaml:"capacity,omitempty"`
	Service  Range   `yaml:"service"`
	Arrival  *Range  `yaml:"arrival,omitempty"`
	Routes   []Route `yaml:"routes,omitempty"`
}

type FirstArrival struct {
	Queue string  `yaml:"queue"`
	Time  float64 `yaml:"time"`
}

type Simulation struct {
	Seeds        []uint64     `yaml:"seeds"`
	RNGPerSeed   int          `yaml:"rngPerSeed"`
	FirstArrival FirstArrival `yaml:"firstArrival"`
}

type File struct {
	Simulation Simulation `yaml:"simulation"`
	Queues     []Queue    `yaml:"queues"`
}

func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f File
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := f.validate(); err != nil {
		return nil, err
	}
	return &f, nil
}

func (f *File) validate() error {
	if len(f.Queues) == 0 {
		return fmt.Errorf("no queues defined")
	}
	if len(f.Simulation.Seeds) == 0 {
		return fmt.Errorf("simulation.seeds is empty")
	}
	if f.Simulation.RNGPerSeed <= 0 {
		return fmt.Errorf("simulation.rngPerSeed must be > 0")
	}
	ids := make(map[string]bool, len(f.Queues))
	for _, q := range f.Queues {
		if q.ID == "" {
			return fmt.Errorf("queue with empty id")
		}
		if ids[q.ID] {
			return fmt.Errorf("duplicate queue id %q", q.ID)
		}
		ids[q.ID] = true
	}
	for _, q := range f.Queues {
		for _, r := range q.Routes {
			if !ids[r.To] {
				return fmt.Errorf("queue %q routes to unknown queue %q", q.ID, r.To)
			}
		}
	}
	if !ids[f.Simulation.FirstArrival.Queue] {
		return fmt.Errorf("firstArrival.queue %q is not a defined queue",
			f.Simulation.FirstArrival.Queue)
	}
	return nil
}

func (f *File) ToQueueConfigs() []queue.Config {
	out := make([]queue.Config, 0, len(f.Queues))
	for _, q := range f.Queues {
		cap := -1
		if q.Capacity != nil {
			cap = *q.Capacity
		}
		c := queue.Config{
			ID:         q.ID,
			Servers:    q.Servers,
			Capacity:   cap,
			ServiceMin: q.Service.Min,
			ServiceMax: q.Service.Max,
		}
		if q.Arrival != nil {
			c.HasExternal = true
			c.ArrivalMin = q.Arrival.Min
			c.ArrivalMax = q.Arrival.Max
		}
		for _, r := range q.Routes {
			c.Routes = append(c.Routes, queue.Route{ID: r.To, Probability: r.Probability})
		}
		out = append(out, c)
	}
	return out
}
