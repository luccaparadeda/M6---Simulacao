package queue

type Route struct {
	ID          string
	Probability float64
}

type Config struct {
	ID          string
	Servers     int
	Capacity    int
	ServiceMin  float64
	ServiceMax  float64
	ArrivalMin  float64
	ArrivalMax  float64
	HasExternal bool
	Routes      []Route
}

type Queue struct {
	Cfg        Config
	Population int
	StateTimes map[int]float64
	Losses     int
}

func New(cfg Config) *Queue {
	return &Queue{
		Cfg:        cfg,
		StateTimes: make(map[int]float64),
	}
}

func (q *Queue) CanAccept() bool {
	if q.Cfg.Capacity < 0 {
		return true
	}
	return q.Population < q.Cfg.Capacity
}

func (q *Queue) HasFreeServer() bool {
	return q.InService() < q.Cfg.Servers
}

func (q *Queue) InService() int {
	if q.Population < q.Cfg.Servers {
		return q.Population
	}
	return q.Cfg.Servers
}

func (q *Queue) Accumulate(dt float64) {
	if dt <= 0 {
		return
	}
	q.StateTimes[q.Population] += dt
}
