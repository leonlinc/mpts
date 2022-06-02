package ts

import (
	"io"

	"github.com/leonlinc/mpts/internal/bit"
)

type adaptation_field struct {
	adaptation_field_length              int
	discontinuity_indicator              bool
	random_access_indicator              bool
	elementary_stream_priority_indicator bool
	PCR_flag                             bool
	OPCR_flag                            bool
	splicing_point_flag                  bool
	transport_private_data_flag          bool
	adaptation_field_extension_flag      bool
	program_clock_reference_base         int64
	program_clock_reference_extension    int64
}

type Reader struct {
	// transport_packet()
	sync_byte                    int
	transport_error_indicator    int
	payload_unit_start_indicator int
	transport_priority           int
	PID                          int
	transport_scrambling_control int
	adaptation_field_control     int
	continuity_counter           int
	// adaptation_field()
	*adaptation_field

	rd         io.Reader
	pos        int64
	PcrRecords map[int]PcrRecord
}

func (r *Reader) Read(b []byte) (n int, err error) {
	n, err = r.rd.Read(b)
	if err != nil {
		return
	}

	br := bit.NewReader(b)
	r.pos += 1
	r.sync_byte = br.ReadBit(8)
	r.transport_error_indicator = br.ReadBit(1)
	r.payload_unit_start_indicator = br.ReadBit(1)
	r.transport_priority = br.ReadBit(1)
	r.PID = br.ReadBit(13)
	r.transport_scrambling_control = br.ReadBit(2)
	r.adaptation_field_control = br.ReadBit(2)
	r.continuity_counter = br.ReadBit(4)
	if r.adaptation_field_control == 2 || r.adaptation_field_control == 3 {
		r.adaptation_field = &adaptation_field{}
		r.adaptation_field_length = br.ReadBit(8)
		if r.adaptation_field_length > 0 {
			r.discontinuity_indicator = (br.ReadBit(1) != 0)
			r.random_access_indicator = (br.ReadBit(1) != 0)
			r.elementary_stream_priority_indicator = (br.ReadBit(1) != 0)
			r.PCR_flag = (br.ReadBit(1) != 0)
			r.OPCR_flag = (br.ReadBit(1) != 0)
			r.splicing_point_flag = (br.ReadBit(1) != 0)
			r.transport_private_data_flag = (br.ReadBit(1) != 0)
			r.adaptation_field_extension_flag = (br.ReadBit(1) != 0)
			if r.PCR_flag {
				r.program_clock_reference_base = br.ReadBit64(33)
				br.SkipBit(6)
				r.program_clock_reference_extension = br.ReadBit64(9)
				pcr := ComputePcr(r.program_clock_reference_base, r.program_clock_reference_extension)
				r.PcrRecords[r.PID] = PcrRecord{Pid: r.PID, Pos: r.pos, Pcr: pcr}
			}
		}
	} else {
		r.adaptation_field = nil
	}
	return
}

func (r *Reader) Pos() int64 {
	return r.pos
}

func (r *Reader) Pcr() (PcrRecord, bool) {
	return r.PcrRecords[r.PID], r.adaptation_field != nil && r.PCR_flag
}

func NewReader(r io.Reader) *Reader {
	return &Reader{rd: r, PcrRecords: make(map[int]PcrRecord)}
}
