package sim

import (
	"fmt"
	"sort"

	"queuesim/logger"
	"queuesim/queue"
	"queuesim/rng"
	"queuesim/scheduler"
)

type Router interface {
	Next(fromQueueID string, r *rng.LCG) (string, bool)
}

type ProbabilityRouter struct {
	Routes map[string][]queue.Route
}

func NewProbabilityRouter(queues []*queue.Queue) *ProbabilityRouter {
	m := make(map[string][]queue.Route)
	for _, q := range queues {
		m[q.Cfg.ID] = q.Cfg.Routes
	}
	return &ProbabilityRouter{Routes: m}
}

func (p *ProbabilityRouter) Next(from string, r *rng.LCG) (string, bool) {
	routes := p.Routes[from]
	if len(routes) == 0 {
		return "", true
	}
	u, ok := r.NextUniform()
	if !ok {
		return "", false
	}
	acc := 0.0
	for _, rt := range routes {
		acc += rt.Probability
		if u < acc {
			return rt.ID, true
		}
	}
	return "", true
}

type Simulator struct {
	Queues    map[string]*queue.Queue
	Order     []string
	Sched     *scheduler.Scheduler
	Rng       *rng.LCG
	Router    Router
	Logger    logger.Logger
	Clock     float64
	lastClock float64
}

func New(configs []queue.Config, r *rng.LCG) *Simulator {
	qs := make(map[string]*queue.Queue)
	order := make([]string, 0, len(configs))
	qList := make([]*queue.Queue, 0, len(configs))
	for _, c := range configs {
		q := queue.New(c)
		qs[c.ID] = q
		order = append(order, c.ID)
		qList = append(qList, q)
	}
	return &Simulator{
		Queues: qs,
		Order:  order,
		Sched:  scheduler.New(),
		Rng:    r,
		Router: NewProbabilityRouter(qList),
	}
}

func (s *Simulator) ScheduleFirstArrival(queueID string, t0 float64) {
	s.Sched.Schedule(&scheduler.Event{Time: t0, Kind: scheduler.Arrival, QueueID: queueID})
}

func (s *Simulator) accumulateAll(now float64) {
	dt := now - s.lastClock
	for _, id := range s.Order {
		s.Queues[id].Accumulate(dt)
	}
	s.lastClock = now
}

func (s *Simulator) snapshot() (map[string]int, map[string]int) {
	pops := make(map[string]int, len(s.Order))
	losses := make(map[string]int, len(s.Order))
	for _, id := range s.Order {
		pops[id] = s.Queues[id].Population
		losses[id] = s.Queues[id].Losses
	}
	return pops, losses
}

func (s *Simulator) log(event, queueID, detail string) {
	if s.Logger == nil {
		return
	}
	pops, losses := s.snapshot()
	s.Logger.Log(logger.Entry{
		RNGCount:    s.Rng.Count(),
		Time:        s.Clock,
		Event:       event,
		QueueID:     queueID,
		Detail:      detail,
		Populations: pops,
		Losses:      losses,
	})
}

func (s *Simulator) Run() {
	for {
		if s.Rng.Exhausted() {
			s.log("STOP", "", "RNG budget exhausted")
			return
		}
		ev := s.Sched.Next()
		if ev == nil {
			return
		}
		s.accumulateAll(ev.Time)
		s.Clock = ev.Time

		switch ev.Kind {
		case scheduler.Arrival:
			if !s.handleArrival(s.Queues[ev.QueueID]) {
				s.log("STOP", ev.QueueID, "RNG budget exhausted during arrival")
				return
			}
		case scheduler.Departure:
			if !s.handleDeparture(s.Queues[ev.QueueID]) {
				s.log("STOP", ev.QueueID, "RNG budget exhausted during departure")
				return
			}
		}
	}
}

func (s *Simulator) handleArrival(q *queue.Queue) bool {
	if !q.CanAccept() {
		q.Losses++
		s.log("LOSS", q.Cfg.ID, fmt.Sprintf("queue full (cap=%d)", q.Cfg.Capacity))
	} else {
		needsService := q.HasFreeServer()
		var serviceDur float64
		if needsService {
			d, ok := s.Rng.Between(q.Cfg.ServiceMin, q.Cfg.ServiceMax)
			if !ok {
				return false
			}
			serviceDur = d
		}
		q.Population++
		if needsService {
			s.Sched.Schedule(&scheduler.Event{
				Time:    s.Clock + serviceDur,
				Kind:    scheduler.Departure,
				QueueID: q.Cfg.ID,
			})
			s.log("ARRIVAL", q.Cfg.ID, fmt.Sprintf("admitted, service starts (dur=%.4f)", serviceDur))
		} else {
			s.log("ARRIVAL", q.Cfg.ID, "admitted, waiting in queue")
		}
	}

	if q.Cfg.HasExternal {
		d, ok := s.Rng.Between(q.Cfg.ArrivalMin, q.Cfg.ArrivalMax)
		if !ok {
			return false
		}
		s.Sched.Schedule(&scheduler.Event{
			Time:    s.Clock + d,
			Kind:    scheduler.Arrival,
			QueueID: q.Cfg.ID,
		})
		s.log("SCHEDULE", q.Cfg.ID, fmt.Sprintf("next arrival at %.4f", s.Clock+d))
	}
	return true
}

func (s *Simulator) handleDeparture(q *queue.Queue) bool {
	routeID, ok := s.Router.Next(q.Cfg.ID, s.Rng)
	if !ok {
		return false
	}

	ownNeedsNext := q.Population-1 >= q.Cfg.Servers
	var ownNextDur float64
	if ownNeedsNext {
		d, ok := s.Rng.Between(q.Cfg.ServiceMin, q.Cfg.ServiceMax)
		if !ok {
			return false
		}
		ownNextDur = d
	}

	var dest *queue.Queue
	destWillLoss := false
	destStartsService := false
	var destDur float64
	if routeID != "" {
		dest = s.Queues[routeID]
		if !dest.CanAccept() {
			destWillLoss = true
		} else if dest.HasFreeServer() {
			destStartsService = true
			d, ok := s.Rng.Between(dest.Cfg.ServiceMin, dest.Cfg.ServiceMax)
			if !ok {
				return false
			}
			destDur = d
		}
	}

	q.Population--
	s.log("DEPARTURE", q.Cfg.ID, fmt.Sprintf("service complete, route -> %s", routeLabel(routeID)))

	if ownNeedsNext {
		s.Sched.Schedule(&scheduler.Event{
			Time:    s.Clock + ownNextDur,
			Kind:    scheduler.Departure,
			QueueID: q.Cfg.ID,
		})
		s.log("SCHEDULE", q.Cfg.ID, fmt.Sprintf("next service ends at %.4f", s.Clock+ownNextDur))
	}

	if dest != nil {
		if destWillLoss {
			dest.Losses++
			s.log("LOSS", dest.Cfg.ID, fmt.Sprintf("routed from %s but queue full (cap=%d)", q.Cfg.ID, dest.Cfg.Capacity))
		} else {
			dest.Population++
			if destStartsService {
				s.Sched.Schedule(&scheduler.Event{
					Time:    s.Clock + destDur,
					Kind:    scheduler.Departure,
					QueueID: dest.Cfg.ID,
				})
				s.log("ARRIVAL", dest.Cfg.ID, fmt.Sprintf("routed from %s, service starts (dur=%.4f)", q.Cfg.ID, destDur))
			} else {
				s.log("ARRIVAL", dest.Cfg.ID, fmt.Sprintf("routed from %s, waiting in queue", q.Cfg.ID))
			}
		}
	}
	return true
}

func routeLabel(id string) string {
	if id == "" {
		return "EXIT"
	}
	return id
}

func (s *Simulator) Report() {
	fmt.Println("============================================================")
	fmt.Println("  RELATÓRIO FINAL DA SIMULAÇÃO")
	fmt.Println("============================================================")
	fmt.Printf("Tempo global total: %.4f\n", s.Clock)
	fmt.Printf("Números aleatórios consumidos: %d\n\n", s.Rng.Count())

	for _, id := range s.Order {
		q := s.Queues[id]
		fmt.Printf("---- Fila %s (G/G/%d/%s) ----\n",
			id, q.Cfg.Servers, capStr(q.Cfg.Capacity))
		fmt.Printf("  Perdas: %d\n", q.Losses)

		total := 0.0
		states := make([]int, 0, len(q.StateTimes))
		for k := range q.StateTimes {
			states = append(states, k)
			total += q.StateTimes[k]
		}
		sort.Ints(states)

		fmt.Printf("  %-8s %-15s %-15s\n", "Estado", "Tempo Acum.", "Probabilidade")
		for _, st := range states {
			t := q.StateTimes[st]
			p := 0.0
			if total > 0 {
				p = t / total
			}
			fmt.Printf("  %-8d %-15.4f %-15.6f\n", st, t, p)
		}
		fmt.Printf("  Tempo total observado: %.4f\n\n", total)
	}
}

func capStr(c int) string {
	if c < 0 {
		return "inf"
	}
	return fmt.Sprintf("%d", c)
}
