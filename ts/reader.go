package ts

import (
	"github.com/leonlinc/mpts/bit"
	"os"
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
	sync_byte                    int
	transport_error_indicator    int
	payload_unit_start_indicator int
	transport_priority           int
	PID                          int
	transport_scrambling_control int
	adaptation_field_control     int
	continuity_counter           int
	*adaptation_field

	file  *os.File
	count int64
}

func (r *Reader) Read(b []byte) (n int, err error) {
	n, err = r.file.Read(b)
	if err == nil {
		k := bit.NewReader(b)

		r.sync_byte = k.ReadBit(8)
		r.transport_error_indicator = k.ReadBit(1)
		r.payload_unit_start_indicator = k.ReadBit(1)
		r.transport_priority = k.ReadBit(1)
		r.PID = k.ReadBit(13)
		r.transport_scrambling_control = k.ReadBit(2)
		r.adaptation_field_control = k.ReadBit(2)
		r.continuity_counter = k.ReadBit(4)
		if r.adaptation_field_control == 2 || r.adaptation_field_control == 3 {
			r.adaptation_field = &adaptation_field{}
			r.adaptation_field_length = k.ReadBit(8)
			if r.adaptation_field_length > 0 {
				r.discontinuity_indicator = (k.ReadBit(1) != 0)
				r.random_access_indicator = (k.ReadBit(1) != 0)
				r.elementary_stream_priority_indicator = (k.ReadBit(1) != 0)
				r.PCR_flag = (k.ReadBit(1) != 0)
				r.OPCR_flag = (k.ReadBit(1) != 0)
				r.splicing_point_flag = (k.ReadBit(1) != 0)
				r.transport_private_data_flag = (k.ReadBit(1) != 0)
				r.adaptation_field_extension_flag = (k.ReadBit(1) != 0)
				if r.PCR_flag {
					r.program_clock_reference_base = k.ReadBit64(33)
					k.SkipBit(6)
					r.program_clock_reference_extension = k.ReadBit64(9)
				}
			}
		} else {
			r.adaptation_field = nil
		}
		r.count += 1
	}
	return
}

func (r *Reader) Count() int64 {
	return r.count
}

func (r *Reader) PCR() int64 {
	if r.adaptation_field != nil && r.PCR_flag {
		return r.program_clock_reference_base*300 + r.program_clock_reference_extension
	} else {
		return -1
	}
}

func NewReader(file *os.File) *Reader {
	return &Reader{file: file}
}
