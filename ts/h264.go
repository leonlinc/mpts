package ts

import (
	"bytes"
)

var NalUnitType []string = []string{
	"unspecified",
	"slice_non_idr",
	"slice_partition_A",
	"slice_partition_B",
	"slice_partition_C",
	"slice_idr",
	"sei",
	"seq_param_set",
	"pic_param_set",
	"au_delimiter",
	"end_of_seq",
	"end_of_stream",
	"filler_data",
	"seq_param_set_ext",
	"prefix",
	"sub_seq_param_set",
}

func GetNalUnitType(b int) string {
	b = b & 0x1F
	if b < len(NalUnitType) {
		return NalUnitType[b]
	} else {
		return NalUnitType[0]
	}
}

func ParseNalUnits(data []byte) []string {
	var nals []string
	var pos int
	var startcode = []byte{0, 0, 1}
	var startlen = len(startcode)
	for pos+5 < len(data) {
		if bytes.Compare(startcode, data[pos:pos+startlen]) == 0 {
			pos += startlen
			nal := GetNalUnitType(int(data[pos]))
			nals = append(nals, nal)
		}
		pos += 1
	}
	return nals
}
