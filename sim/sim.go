package sim

import (
	"fmt"
	"sort"

	"queuesim/queue"
	"queuesim/rng"
	"queuesim/scheduler"
)

// Router decide o destino de um cliente ao sair de uma fila.
// SEMPRE consome 1 número aleatório (mesmo com rota determinística),
// conforme convenção acadêmica de simulação discreta.
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

// Next sorteia U(0,1) e percorre as probabilidades acumuladas.
// Se a fila não tem rotas configuradas, NÃO consome RNG (saída imediata).
// Caso contrário, consome 1 RNG independentemente das probabilidades.
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
	return "", true // probabilidade residual => saída do sistema
}

type Simulator struct {
	Queues    map[string]*queue.Queue
	Order     []string
	Sched     *scheduler.Scheduler
	Rng       *rng.LCG
	Router    Router
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

// Run executa até que o RNG tenha sido esgotado (100.000º número consumido).
// Semântica: se um evento em andamento consumir o 100.000º número, esse evento
// termina sua mutação de estado normalmente; o próximo evento NÃO é iniciado.
// Sampling é feito ANTES de mutar estado para evitar inconsistências.
func (s *Simulator) Run() {
	for {
		if s.Rng.Exhausted() {
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
				return
			}
		case scheduler.Departure:
			if !s.handleDeparture(s.Queues[ev.QueueID]) {
				return
			}
		}
	}
}

// handleArrival processa chegada externa. Retorna false se o RNG esgotou
// antes de completar o sampling necessário (estado não é mutado nesse caso).
func (s *Simulator) handleArrival(q *queue.Queue) bool {
	// 1) Pré-sampling de tudo que o evento precisa.
	if !q.CanAccept() {
		q.Losses++
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
		}
	}
	// 2) Agendar próxima chegada externa (se aplicável).
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
	}
	return true
}

// handleDeparture processa fim de serviço: pré-sampla rota, próximo serviço
// desta fila (se há backlog) e serviço no destino (se aceita e tem servidor livre),
// e só então muta estado.
func (s *Simulator) handleDeparture(q *queue.Queue) bool {
	// 1) Rota (sempre consome 1 RNG se há rotas configuradas).
	routeID, ok := s.Router.Next(q.Cfg.ID, s.Rng)
	if !ok {
		return false
	}

	// 2) Próximo serviço desta fila, se ainda há cliente esperando após a saída.
	ownNeedsNext := q.Population-1 >= q.Cfg.Servers
	var ownNextDur float64
	if ownNeedsNext {
		d, ok := s.Rng.Between(q.Cfg.ServiceMin, q.Cfg.ServiceMax)
		if !ok {
			return false
		}
		ownNextDur = d
	}

	// 3) Transição para destino (se houver rota interna).
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

	// 4) Mutações atômicas (todos os RNGs já foram obtidos com sucesso).
	q.Population--
	if ownNeedsNext {
		s.Sched.Schedule(&scheduler.Event{
			Time:    s.Clock + ownNextDur,
			Kind:    scheduler.Departure,
			QueueID: q.Cfg.ID,
		})
	}
	if dest != nil {
		if destWillLoss {
			dest.Losses++
		} else {
			dest.Population++
			if destStartsService {
				s.Sched.Schedule(&scheduler.Event{
					Time:    s.Clock + destDur,
					Kind:    scheduler.Departure,
					QueueID: dest.Cfg.ID,
				})
			}
		}
	}
	return true
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
