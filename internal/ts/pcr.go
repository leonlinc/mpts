package ts

type PcrRecord struct {
	Pid int
	Pos int64
	Pcr int64
}

func ComputePcr(base, ext int64) int64 {
	return base*300 + ext
}
