package mpts

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type SpliceInfoSection struct {
	table_id                 int
	section_syntax_indicator int
	private_indicator        int
	reserved                 int
	section_length           int
	protocol_version         int
	encrypted_packet         int
	encryption_algorithm     int
	pts_adjustment           int64
	cw_index                 int
	tier                     int
	splice_command_length    int
	splice_command_type      int
	descriptor_loop_length   int
	SpliceDescriptorList     []SpliceDescriptor
	*SpliceInsert
	*TimeSignal
	*SegmentDescriptor
}

type SpliceInsert struct {
	splice_event_id               int64
	splice_event_cancel_indicator int
	reserved                      int
	out_of_network_indicator      int
	program_splice_flag           int
	duration_flag                 int
	splice_immediate_flag         int
	component_count               int
	component_tag                 int
	unique_program_id             int
	avail_num                     int
	avails_expected               int
	*SpliceTime
	*BreakDuration
}

type TimeSignal struct {
	SpliceTime
}

type SpliceTime struct {
	time_specified_flag int
	reserved            int
	pts_time            int64
}

type BreakDuration struct {
	auto_return int
	reserved    int
	duration    int64
}

type SpliceDescriptor struct {
	splice_descriptor_tag int
	descriptor_length     int
	identifier            int
}

type SegmentDescriptor struct {
	segmentation_event_id               int
	segmentation_event_cancel_indicator int
	reserved                            int
	program_segmentation_flag           int
	segmentation_duration_flag          int
	delivery_not_restricted_flag        int
	web_delivery_allowed_flag           int
	no_regional_blackout_flag           int
	archive_allowed_flag                int
	device_restrictions                 int
	component_count                     int
	component_tag                       int
	pts_offset                          int64
	segmentation_duration               int64
	segmentation_upid_type              int
	segmentation_upid_length            int
	segmentation_type_id                int
	segment_num                         int
	segments_expected                   int
}

func ParseSpliceInfoSection(data []byte) *SpliceInfoSection {
	r := NewReader(data)
	pointer := r.ReadBit(8)
	r.SkipByte(pointer)

	section := &SpliceInfoSection{}
	section.table_id = r.ReadBit(8)
	section.section_syntax_indicator = r.ReadBit(1)
	section.private_indicator = r.ReadBit(1)
	section.reserved = r.ReadBit(2)
	section.section_length = r.ReadBit(12)
	section.protocol_version = r.ReadBit(8)
	section.encrypted_packet = r.ReadBit(1)
	section.encryption_algorithm = r.ReadBit(6)
	section.pts_adjustment = r.ReadBit64(33)
	section.cw_index = r.ReadBit(8)
	section.tier = r.ReadBit(12)
	section.splice_command_length = r.ReadBit(12)
	section.splice_command_type = r.ReadBit(8)

	switch section.splice_command_type {
	case 5:
		section.SpliceInsert = ParseSpliceInsert(r)
	case 6:
		section.TimeSignal = ParseTimeSignal(r)
	default:
		r.SkipByte(section.splice_command_length)
	}

	section.descriptor_loop_length = r.ReadBit(16)
	descriptor_loop_length := section.descriptor_loop_length
	for descriptor_loop_length > 0 {
		descriptor_length := ParseSpliceDescriptor(section, r)
		descriptor_loop_length -= descriptor_length
	}

	return section
}

func ParseSpliceInsert(r *Reader) *SpliceInsert {
	insert := &SpliceInsert{}
	insert.splice_event_id = r.ReadBit64(32)
	insert.splice_event_cancel_indicator = r.ReadBit(1)
	insert.reserved = r.ReadBit(7)
	if insert.splice_event_cancel_indicator == 0 {
		insert.out_of_network_indicator = r.ReadBit(1)
		insert.program_splice_flag = r.ReadBit(1)
		insert.duration_flag = r.ReadBit(1)
		insert.splice_immediate_flag = r.ReadBit(1)
		insert.reserved = r.ReadBit(4)
		if insert.program_splice_flag == 1 && insert.splice_immediate_flag == 0 {
			insert.SpliceTime = ParseSpliceTime(r)
		}
		if insert.program_splice_flag == 0 {
			insert.component_count = r.ReadBit(8)
			for i := 0; i < insert.component_count; i++ {
				insert.component_tag = r.ReadBit(8)
				if insert.splice_immediate_flag == 0 {
					insert.SpliceTime = ParseSpliceTime(r)
				}
			}
		}
		if insert.duration_flag == 1 {
			insert.BreakDuration = ParseBreakDuration(r)
		}
		insert.unique_program_id = r.ReadBit(16)
		insert.avail_num = r.ReadBit(8)
		insert.avails_expected = r.ReadBit(8)
	}
	return insert
}

func ParseTimeSignal(r *Reader) *TimeSignal {
	signal := &TimeSignal{}
	signal.SpliceTime = *ParseSpliceTime(r)
	return signal
}

func ParseSpliceTime(r *Reader) *SpliceTime {
	time := &SpliceTime{}
	time.time_specified_flag = r.ReadBit(1)
	if time.time_specified_flag == 1 {
		time.reserved = r.ReadBit(6)
		time.pts_time = r.ReadBit64(33)
	} else {
		time.reserved = r.ReadBit(7)
	}
	return time
}

func ParseBreakDuration(r *Reader) *BreakDuration {
	duration := &BreakDuration{}
	duration.auto_return = r.ReadBit(1)
	duration.reserved = r.ReadBit(6)
	duration.duration = r.ReadBit64(33)
	return duration
}

func ParseSpliceDescriptor(section *SpliceInfoSection, r *Reader) int {
	descriptor := SpliceDescriptor{}
	descriptor.splice_descriptor_tag = r.ReadBit(8)
	descriptor.descriptor_length = r.ReadBit(8)
	descriptor.identifier = r.ReadBit(32)
	section.SpliceDescriptorList = append(section.SpliceDescriptorList, descriptor)

	if descriptor.splice_descriptor_tag == 2 {
		section.SegmentDescriptor = ParseSegmentDescriptor(r)
	} else {
		r.SkipByte(descriptor.descriptor_length - 4)
	}

	return descriptor.descriptor_length + 2
}

func ParseSegmentDescriptor(r *Reader) *SegmentDescriptor {
	segment := &SegmentDescriptor{}
	segment.segmentation_event_id = r.ReadBit(32)
	segment.segmentation_event_cancel_indicator = r.ReadBit(1)
	segment.reserved = r.ReadBit(7)
	if segment.segmentation_event_cancel_indicator == 0 {
		segment.program_segmentation_flag = r.ReadBit(1)
		segment.segmentation_duration_flag = r.ReadBit(1)
		segment.delivery_not_restricted_flag = r.ReadBit(1)
		if segment.delivery_not_restricted_flag == 0 {
			segment.web_delivery_allowed_flag = r.ReadBit(1)
			segment.no_regional_blackout_flag = r.ReadBit(1)
			segment.archive_allowed_flag = r.ReadBit(1)
			segment.device_restrictions = r.ReadBit(2)
		} else {
			segment.reserved = r.ReadBit(5)
		}
		if segment.program_segmentation_flag == 0 {
			segment.component_count = r.ReadBit(8)
			for i := 0; i < segment.component_count; i++ {
				segment.component_tag = r.ReadBit(8)
				segment.reserved = r.ReadBit(7)
				segment.pts_offset = r.ReadBit64(33)
			}
		}
		if segment.segmentation_duration_flag == 1 {
			r.SkipBit(7)
			segment.segmentation_duration = r.ReadBit64(33)
		}
		segment.segmentation_upid_type = r.ReadBit(8)
		segment.segmentation_upid_length = r.ReadBit(8)
		// segmentation_upid()
		r.SkipByte(segment.segmentation_upid_length)
		segment.segmentation_type_id = r.ReadBit(8)
		segment.segment_num = r.ReadBit(8)
		segment.segments_expected = r.ReadBit(8)
	}
	return segment
}

func (section SpliceInfoSection) GetSpliceType() string {
	switch section.splice_command_type {
	case 5:
		return "splice_insert"
	case 6:
		return "time_signal"
	default:
		return "private"
	}
}

func (section SpliceInfoSection) GetSpliceTime() (int64, int64) {
	var spliceTime int64 = -1
	switch section.splice_command_type {
	case 5:
		spliceTime = section.SpliceInsert.GetSpliceTime()
	case 6:
		spliceTime = section.TimeSignal.GetSpliceTime()
	}
	return spliceTime, section.pts_adjustment
}

func (section SpliceInfoSection) GetSpliceDuration() int64 {
	var duration int64 = -1
	switch section.splice_command_type {
	case 5:
		duration = section.SpliceInsert.GetSpliceDuration()
	case 6:
		if section.SegmentDescriptor != nil {
			return section.SegmentDescriptor.GetSpliceDuration()
		}
	}
	return duration
}

func (section SpliceInfoSection) GetSegType() int {
	var segType int = -1
	switch section.splice_command_type {
	case 5:
		// 0: splice-in, 1: splice-out
		segType = section.SpliceInsert.GetOutOfNetworkIndicator()
	case 6:
		// segmentation_type_id
		if section.SegmentDescriptor != nil {
			segType = section.SegmentDescriptor.GetSegType()
		}
	}
	return segType
}

func (insert SpliceInsert) GetSpliceTime() int64 {
	if insert.SpliceTime != nil {
		if insert.SpliceTime.time_specified_flag == 1 {
			return insert.SpliceTime.pts_time
		}
	}
	return -1
}

func (insert SpliceInsert) GetSpliceDuration() int64 {
	if insert.BreakDuration != nil {
		return insert.BreakDuration.duration
	}
	return -1
}

func (insert SpliceInsert) GetOutOfNetworkIndicator() int {
	return insert.out_of_network_indicator
}

func (signal TimeSignal) GetSpliceTime() int64 {
	if signal.SpliceTime.time_specified_flag == 1 {
		return signal.SpliceTime.pts_time
	}
	return -1
}

func (segment SegmentDescriptor) GetSpliceDuration() int64 {
	if segment.segmentation_event_cancel_indicator == 0 {
		if segment.segmentation_duration_flag == 1 {
			return segment.segmentation_duration
		}
	}
	return -1
}

func (segment SegmentDescriptor) GetSegType() int {
	return segment.segmentation_type_id
}

type Scte35Record struct {
	BaseRecord
	CurByte  int64
	CurTime  int64
	CurData  []byte
	BytePos  []int64
	PcrTime  []int64
	Sections []*SpliceInfoSection
}

func (s *Scte35Record) Process(pkt *TsPkt) {
	if pkt.PUSI == 1 {
		if s.CurData != nil {
			section := ParseSpliceInfoSection(s.CurData)
			s.BytePos = append(s.BytePos, s.CurByte)
			s.PcrTime = append(s.PcrTime, s.CurTime)
			s.Sections = append(s.Sections, section)
		}
		s.CurByte = pkt.Pos
		s.CurTime = s.BaseRecord.PcrTime
		s.CurData = pkt.Data
	} else {
		s.CurData = append(s.CurData, pkt.Data...)
	}
}

func (s *Scte35Record) Flush() {
	if s.CurData != nil {
		section := ParseSpliceInfoSection(s.CurData)
		s.BytePos = append(s.BytePos, s.CurByte)
		s.PcrTime = append(s.PcrTime, s.CurTime)
		s.Sections = append(s.Sections, section)
	}
}

func (s *Scte35Record) Report(root string) {
	fname := filepath.Join(root, strconv.Itoa(s.Pid)+".csv")
	w, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	defer w.Close()

	fmt.Fprintln(w, "pos, pcr, type, pts_time, pts_adjust, duration, out_or_segType")
	if s.Sections != nil {
		for i, section := range s.Sections {
			splice_type := section.GetSpliceType()
			pts, adj := section.GetSpliceTime()
			duration := section.GetSpliceDuration()
			segType := section.GetSegType()
			fmt.Fprintf(w, "%v, %v, %v, %v, %v, %v, %v\n",
				s.BytePos[i],
				s.PcrTime[i]/300,
				splice_type,
				pts,
				adj,
				duration,
				segType)
		}
	}
}
