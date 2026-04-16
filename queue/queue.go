package queue

// Route descreve um destino de encaminhamento com sua probabilidade.
// Se ID == "" representa saída do sistema (exit).
type Route struct {
	ID          string
	Probability float64
}

// Config parametriza uma fila G/G/c/K.
type Config struct {
	ID           string
	Servers      int
	Capacity     int // -1 = infinita
	ServiceMin   float64
	ServiceMax   float64
	ArrivalMin   float64 // usado apenas se for fila de entrada externa
	ArrivalMax   float64
	HasExternal  bool
	Routes       []Route // soma das probabilidades <= 1; resto vira saída
}

// Queue representa o estado dinâmico de uma fila.
type Queue struct {
	Cfg         Config
	Population  int              // clientes na fila (em espera + em serviço)
	StateTimes  map[int]float64  // tempo acumulado em cada estado (população)
	Losses      int
}

func New(cfg Config) *Queue {
	return &Queue{
		Cfg:        cfg,
		StateTimes: make(map[int]float64),
	}
}

// CanAccept indica se a fila aceita mais um cliente.
func (q *Queue) CanAccept() bool {
	if q.Cfg.Capacity < 0 {
		return true
	}
	return q.Population < q.Cfg.Capacity
}

// HasFreeServer indica se há servidor disponível (para iniciar serviço imediato).
func (q *Queue) HasFreeServer() bool {
	return q.InService() < q.Cfg.Servers
}

// InService devolve quantos estão sendo atendidos.
func (q *Queue) InService() int {
	if q.Population < q.Cfg.Servers {
		return q.Population
	}
	return q.Cfg.Servers
}

// Accumulate adiciona dt ao estado atual.
func (q *Queue) Accumulate(dt float64) {
	if dt <= 0 {
		return
	}
	q.StateTimes[q.Population] += dt
}
