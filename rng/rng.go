package rng

// LCG é um gerador congruente linear simples.
// Xn+1 = (a*Xn + c) mod M
type LCG struct {
	a, c, m uint64
	state   uint64
	count   int
	limit   int
}

func NewLCG(seed uint64, limit int) *LCG {
	return &LCG{
		a:     1664525,
		c:     1013904223,
		m:     1 << 32,
		state: seed,
		limit: limit,
	}
}

// Count retorna quantos números já foram consumidos.
func (g *LCG) Count() int { return g.count }

// Exhausted indica se o limite global de números aleatórios foi atingido.
func (g *LCG) Exhausted() bool { return g.count >= g.limit }

// NextUniform retorna um U(0,1). Retorna (0, false) se o limite foi atingido.
func (g *LCG) NextUniform() (float64, bool) {
	if g.Exhausted() {
		return 0, false
	}
	g.state = (g.a*g.state + g.c) % g.m
	g.count++
	return float64(g.state) / float64(g.m), true
}

// Between retorna U(a,b) consumindo um número aleatório.
func (g *LCG) Between(a, b float64) (float64, bool) {
	u, ok := g.NextUniform()
	if !ok {
		return 0, false
	}
	return a + (b-a)*u, true
}
