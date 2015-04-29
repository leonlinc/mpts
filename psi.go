package ts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"encoding/hex"
)

var StreamTypeString map[int]string = map[int]string{
	// ISO/IEC 13818-1
	0x00: "Reserved",
	0x01: "MPEG-1 Video",
	0x02: "MPEG-2 Video",
	0x03: "MPEG-1 Audio",
	0x04: "MPEG-2 Audio",
	0x05: "Private Section",
	0x06: "Private PES",
	0x0F: "MPEG-2 AAC Audio (ADTS)",
	0x10: "MPEG-4 Video",
	0x11: "MPEG-4 AAC Audio (LATM)",
	0x1B: "MPEG-4 AVC Video",
	0x81: "AC-3 Audio",
	0x82: "SCTE-27",
	0x86: "SCTE-35",
	0x87: "E-AC-3 Audio",
}

func GetStreamType(s Stream) string {
	if typeString, ok := StreamTypeString[s.stream_type]; ok {
		return typeString
	} else {
		return "Unknown stream type"
	}
}

var DescriptorTagString map[int]string = map[int]string{
	// ISO/IEC 13818-1
	0: "reserved",
	1: "forbidden",
	2: "video_stream_descriptor",
	3: "audio_stream_descriptor",
	4: "hierarchy_descriptor",
	5: "registration_descriptor",
	6: "data_stream_alignment_descriptor",
	7: "target_background_grid_descriptor",
	8: "video_window_descriptor",
	9: "CA_descriptor",
	10: "ISO_639_language_descriptor",
	11: "system_clock_descriptor",
	12: "multiplex_buffer_utilization_descriptor",
	13: "copyright_descriptor",
	14: "maximum_bitrate_descriptor",
	15: "private_data_indicator_descriptor",
	16: "smoothing_buffer_descriptor",
	17: "STD_descriptor",
	18: "IBP_descriptor",
	27: "MPEG-4_video_descriptor",
	28: "MPEG-4_audio_descriptor",
	29: "IOD_descriptor",
	30: "SL_descriptor",
	31: "FMC_descriptor",
	32: "external_ES_ID_descriptor",
	33: "MuxCode_descriptor",
	34: "FmxBufferSize_descriptor",
	35: "multiplexbuffer_descriptor",
	36: "content_labeling_descriptor",
	37: "metadata_pointer_descriptor",
	38: "metadata_descriptor",
	39: "metadata_STD_descriptor",
	40: "AVC video descriptor",
	41: "IPMP_descriptor",
	42: "AVC timing and HRD descriptor",
	43: "MPEG-2_AAC_audio_descriptor",
	44: "FlexMuxTiming_descriptor",
	45: "MPEG-4_text_descriptor",
	46: "MPEG-4_audio_extension_descriptor",
	47: "auxiliary_video_stream_descriptor",
	48: "SVC extension descriptor",
	49: "MVC extension descriptor",
	50: "J2K video descriptor",
	51: "MVC operation point descriptor",
	52: "MPEG2_stereoscopic_video_format_descriptor",
	53: "Stereoscopic_program_info_descriptor",
	54: "Stereoscopic_video_info_descriptor",
	// ETSI EN 300 468
	0x45: "vbi_data_descriptor",
	0x46: "vbi_teletext_descriptor",
	0x56: "teletext_descriptor",
	0x59: "subtitling_descriptor",
	// ATSC A/52
	0x6A: "AC-3_descriptor",              // DVB
	0x81: "AC-3_audio_stream_descriptor", // ATSC
	// Random stuff
	0xDD: "harmonic_aac_bitrate_descriptor",
	0xDE: "harmonic_h264_bitrate_descriptor",
}

func GetDescriptorTabString(tag int) string {
	if tagString, ok := DescriptorTagString[tag]; ok {
		return tagString
	} else {
		return "Unknown descriptor tag"
	}
}

func ParsePat(data []byte) *Pat {
	pat := Pat{}
	r := &Reader{Data: data}
	pointer := r.ReadBit(8)
	r.SkipByte(pointer)
	r.SkipBit(12)
	pat.section_length = r.ReadBit(12)
	pat.transport_stream_id = r.ReadBit(16)
	r.SkipBit(2)
	pat.version_number = r.ReadBit(5)
	pat.current_next_indicator = r.ReadBit(1)
	pat.section_number = r.ReadBit(8)
	pat.last_section_number = r.ReadBit(8)
	section_length := pat.section_length
	// 5 bytes before and 4 bytes after programs
	section_length -= 5 + 4
	pat.programs = make(map[int]Program)
	for section_length > 0 {
		program := Program{}
		program.Number = r.ReadBit(16)
		if program.Number != 0 {
			r.SkipBit(3)
			program.PmtPid = r.ReadBit(13)
			pat.programs[program.PmtPid] = program
		} else {
			r.SkipByte(2)
		}
		section_length -= 4
	}
	pat.crc = r.ReadBit(32)
	return &pat
}

func ParsePmt(data []byte) *Pmt {
	pmt := Pmt{}
	r := &Reader{Data: data}
	pointer := r.ReadBit(8)
	r.SkipByte(pointer)
	r.SkipBit(12)
	pmt.section_length = r.ReadBit(12)
	pmt.program_number = r.ReadBit(16)
	r.SkipBit(2)
	pmt.version_number = r.ReadBit(5)
	pmt.current_next_indicator = r.ReadBit(1)
	pmt.section_number = r.ReadBit(8)
	pmt.last_section_number = r.ReadBit(8)
	r.SkipBit(3)
	pmt.pcr_pid = r.ReadBit(13)
	r.SkipBit(4)
	pmt.program_info_length = r.ReadBit(12)
	r.SkipByte(pmt.program_info_length)
	section_length := pmt.section_length
	section_length -= 9 + pmt.program_info_length + 4
	pmt.streams = make(map[int]Stream)
	for section_length > 0 {
		s := Stream{}
		n := ParseStream(&s, r)
		pmt.streams[s.Pid] = s
		section_length -= n
	}
	pmt.crc = r.ReadBit(32)
	return &pmt
}

func ParseStream(stream *Stream, r *Reader) int {
	stream.stream_type = r.ReadBit(8)
	stream.StreamType = GetStreamType(*stream)
	r.SkipBit(3)
	stream.Pid = r.ReadBit(13)
	r.SkipBit(4)
	stream.es_info_length = r.ReadBit(12)
	es_info_length := stream.es_info_length
	for es_info_length > 0 {
		descriptor_tag := r.ReadBit(8)
		descriptor_length := r.ReadBit(8)
		d := Descriptor{}
		d.Tag = descriptor_tag
		d.TagName = GetDescriptorTabString(d.Tag)
		d.data = r.Data[r.Base : r.Base+descriptor_length]
		d.Data = hex.EncodeToString(d.data)
		stream.Descriptors = append(stream.Descriptors, d)
		es_info_length -= 2 + descriptor_length
		r.SkipByte(descriptor_length)
	}
	return 5 + stream.es_info_length
}

func ParseRegDescriptor(data []byte) RegistrationDescriptor {
	reg := RegistrationDescriptor{}
	reg.format_identifier = string(data[:4])
	return reg
}

func ParseLangDescriptor(data []byte) ISO639LanguageDescriptor {
	lang := ISO639LanguageDescriptor{}
	for i := 0; i < len(data); i += 4 {
		lang.ISO_639_language_code = append(lang.ISO_639_language_code, string(data[i:i+3]))
		lang.audio_type = append(lang.audio_type, int(data[3]))
	}
	return lang
}

type Info struct {
	Programs map[string]Program
}

type Program struct {
	Number  int
	PmtPid  int
	Streams map[string]Stream
}

type Stream struct {
	stream_type    int
	StreamType     string
	Pid            int
	es_info_length int
	Descriptors    []Descriptor
}

type Pat struct {
	table_id                 int
	section_syntax_indicator int
	section_length           int
	transport_stream_id      int
	version_number           int
	current_next_indicator   int
	section_number           int
	last_section_number      int
	crc                      int
	programs                 map[int]Program
}

type Pmt struct {
	table_id                 int
	section_syntax_indicator int
	section_length           int
	program_number           int
	version_number           int
	current_next_indicator   int
	section_number           int
	last_section_number      int
	pcr_pid                  int
	program_info_length      int
	crc                      int
	streams                  map[int]Stream
}

type Descriptor struct {
	Tag  int
	TagName  string
	data []byte
	Data string
}

type RegistrationDescriptor struct {
	format_identifier string
}

type ISO639LanguageDescriptor struct {
	ISO_639_language_code []string
	audio_type            []int
}

func NewPsiParser() *PsiParser {
	return &PsiParser{
		PmtData: make(map[int][]byte),
		Pmts:    make(map[int]*Pmt),
		Strs:    make(map[int]Stream),
		Pcrs:    make(map[int][]int),
	}
}

type PsiParser struct {
	PatData []byte
	Pat     *Pat
	PmtData map[int][]byte
	Pmts    map[int]*Pmt
	Strs    map[int]Stream
	Pcrs    map[int][]int
	Info
}

func (p *PsiParser) Parse(pkt *TsPkt) bool {
	pid := pkt.Pid

	if p.Pat == nil {
		if pid != 0 {
			return false
		}

		if ok := p.BufferData(pkt, &p.PatData); ok {
			pat := ParsePat(p.PatData)
			p.Pat = pat
		}
	} else {
		// Check pmt pid
		if _, ok := p.Pat.programs[pid]; !ok {
			return false
		}

		pmtData := p.PmtData[pid]
		if ok := p.BufferData(pkt, &pmtData); ok {
			pmt := ParsePmt(pmtData)
			p.Pmts[pid] = pmt
		}
		p.PmtData[pid] = pmtData

		// Check if all pmts have been parsed
		if len(p.Pat.programs) == len(p.Pmts) {
			p.ParseDone()
			return true
		}
	}

	return false
}

func (p *PsiParser) ParseDone() {
	for _, pmt := range p.Pmts {
		for _, stream := range pmt.streams {
			p.Strs[stream.Pid] = stream
			p.Pcrs[pmt.pcr_pid] = append(p.Pcrs[pmt.pcr_pid], stream.Pid)
		}
	}
}

func (p *PsiParser) GetStreams() map[int]Stream {
	return p.Strs
}

func (p *PsiParser) GetPcrs() map[int][]int {
	return p.Pcrs
}

func (p *PsiParser) BufferData(pkt *TsPkt, buf *[]byte) bool {
	if pkt.PUSI == 1 {
		if *buf != nil {
			return true
		} else {
			*buf = pkt.Data
		}
	} else {
		if *buf != nil {
			*buf = append(*buf, pkt.Data...)
		}
	}
	return false
}

func (p *PsiParser) Report(root string) {
	fname := filepath.Join(root, "psi.log")
	w, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	defer w.Close()

	printDescriptor := func(stream Stream) {
		if len(stream.Descriptors) == 0 {
			return
		}
		fmt.Fprintf(w, "\t\tdescriptors:\n")
		for _, descriptor := range stream.Descriptors {
			fmt.Fprintf(w, "\t\t\t")
			fmt.Fprintf(w, "(%v) %v", descriptor.Tag, descriptor.TagName)
			switch descriptor.Tag {
			case 0x05:
				reg := ParseRegDescriptor(descriptor.data)
				fmt.Fprintf(w, ": %v", reg.format_identifier)
			case 0x0A:
				lang := ParseLangDescriptor(descriptor.data)
				fmt.Fprintf(w, ": %v", lang.ISO_639_language_code)
			}
			fmt.Fprintf(w, "\n")
		}
	}

	for _, program := range p.Pat.programs {
		pmt := p.Pmts[program.PmtPid]
		fmt.Fprintf(w, "[program] num: %v, pmt: %v, pcr: %v\n",
			program.Number, program.PmtPid, pmt.pcr_pid)
		for _, stream := range pmt.streams {
			fmt.Fprintf(w, "\t")
			fmt.Fprintf(w, "[stream] pid: %v, type: %v\n",
				stream.Pid, GetStreamType(stream))
			printDescriptor(stream)
		}
	}

	jsonFileName := filepath.Join(root, "psi.json")
	jsonFile, err := os.Create(jsonFileName)
	if err != nil {
		panic(err)
	}
	defer jsonFile.Close()
	p.Info.Programs = make(map[string]Program)
	for num, program := range p.Pat.programs {
		program.Streams = make(map[string]Stream)
		p.Info.Programs[strconv.Itoa(num)] = program
		pmt := p.Pmts[program.PmtPid]
		for pid, stream := range pmt.streams {
			program.Streams[strconv.Itoa(pid)] = stream
		}
	}
	buf, _ := json.MarshalIndent(p.Info, "", "  ")
	fmt.Fprintln(jsonFile, string(buf))
}
