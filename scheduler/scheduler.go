package scheduler

import "container/heap"

type EventKind int

const (
	Arrival EventKind = iota // chegada externa
	Departure                 // fim de serviço em uma fila
)

type Event struct {
	Time    float64
	Kind    EventKind
	QueueID string
	index   int
}

type eventHeap []*Event

func (h eventHeap) Len() int            { return len(h) }
func (h eventHeap) Less(i, j int) bool  { return h[i].Time < h[j].Time }
func (h eventHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i]; h[i].index = i; h[j].index = j }
func (h *eventHeap) Push(x interface{}) { e := x.(*Event); e.index = len(*h); *h = append(*h, e) }
func (h *eventHeap) Pop() interface{} {
	old := *h
	n := len(old)
	e := old[n-1]
	*h = old[:n-1]
	return e
}

type Scheduler struct {
	h *eventHeap
}

func New() *Scheduler {
	h := &eventHeap{}
	heap.Init(h)
	return &Scheduler{h: h}
}

func (s *Scheduler) Schedule(e *Event) { heap.Push(s.h, e) }

func (s *Scheduler) Next() *Event {
	if s.h.Len() == 0 {
		return nil
	}
	return heap.Pop(s.h).(*Event)
}

func (s *Scheduler) Len() int { return s.h.Len() }
