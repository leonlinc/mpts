package mpts

const BYTE = 8

type Reader struct {
	Data []byte
	Base int
	Off  int
}

func NewReader(data []byte) *Reader {
	return &Reader{data, 0, 0}
}

func (r *Reader) SkipByte(n int) {
	r.Base += n
}

func (r *Reader) SkipBit(n int) {
	r.Base += n / 8
	r.Off += n % 8
}

func (r *Reader) ReadBit(n int) (v int) {
	return int(r.ReadBit64(n))
}

func (r *Reader) ReadBit64(n int) (v int64) {
	var mask byte
	var sw uint
	for n > 0 {
		if r.Off+n >= BYTE {
			// Read all remaining bits in the current byte
			sw = uint(BYTE - r.Off)
			v <<= sw
			mask = (1<<sw - 1)
			v += int64(mask & r.Data[r.Base])

			n -= BYTE - r.Off
			r.Off = 0
			r.Base++
		} else {
			// Read exactly n sw
			v <<= uint(n)
			sw = uint(BYTE - r.Off)
			mask = (1 << sw)
			sw = uint(BYTE - r.Off - n)
			mask -= (1 << sw)
			v += int64((mask & r.Data[r.Base]) >> sw)

			r.Off += n
			n = 0
		}
	}
	return v
}
