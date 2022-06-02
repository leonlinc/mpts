package ts

type PatChecker struct {
	ppcr int64
	pkts []int64
}

func (p *PatChecker) Check(r *Reader) {
	if r.PID == 0 {
		p.pkts = append(p.pkts, r.Pos())
	}
}

func NewPatChecker() *PatChecker {
	return &PatChecker{ppcr: -1}
}
