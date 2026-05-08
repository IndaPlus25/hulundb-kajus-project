package dns

import (
	"encoding/binary"
	"fmt"
	"math/rand"
)

const (
	maskQR     = 0x8000 // bit 15
	maskOpcode = 0x7800 // bits 14-11
	maskAA     = 0x0400 // bit 10
	maskTC     = 0x0200 // bit 9
	maskRD     = 0x0100 // bit 8
	maskRA     = 0x0080 // bit 7
	maskZ      = 0x0070 // bits 6-4 (reserved, always zero)
	maskRCode  = 0x000F // bits 3-0

	// Shifts for multi-bit fields
	shiftOpcode = 11

	// Opcode values (RFC 1035 §4.1.1)
	OpcodeQuery  uint8 = 0
	OpcodeIQuery uint8 = 1
	OpcodeStatus uint8 = 2

	// RCode values (RFC 1035 §4.1.1)
	RCodeNoError        uint8 = 0
	RCodeFormatError    uint8 = 1
	RCodeServerFailure  uint8 = 2
	RCodeNameError      uint8 = 3 // NXDOMAIN
	RCodeNotImplemented uint8 = 4
	RCodeRefused        uint8 = 5
)

type Header struct {
	ID      uint16
	QR      bool
	Opcode  uint8
	AA      bool
	TC      bool
	RD      bool
	RA      bool
	RCode   uint8
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

func DecodeHeader(b []byte) (Header, error) {

	if len(b) < 12 {
		return Header{}, fmt.Errorf("header too short")
	}

	h := Header{}

	h.ID = binary.BigEndian.Uint16(b[0:2])
	flags := binary.BigEndian.Uint16(b[2:4])

	//Logga error i framtiden
	// if (flags>>4)&0x7 != 0 {
	// 	return Header{}, fmt.Errorf("zero bits are not zero")
	// }

	h.QR = ((flags >> 15) & 0x1) != 0
	h.Opcode = uint8((flags >> shiftOpcode) & 0xF)
	h.AA = ((flags >> 10) & 0x1) != 0
	h.TC = ((flags >> 9) & 0x1) != 0
	h.RD = ((flags >> 8) & 0x1) != 0
	h.RA = ((flags >> 7) & 0x1) != 0
	h.RCode = uint8(flags & maskRCode)

	h.QDCount = binary.BigEndian.Uint16(b[4:6])
	h.ANCount = binary.BigEndian.Uint16(b[6:8])
	h.NSCount = binary.BigEndian.Uint16(b[8:10])
	h.ARCount = binary.BigEndian.Uint16(b[10:12])

	return h, nil
}

func (h Header) Encode() []byte {
	buf := make([]byte, 12)

	// ID
	binary.BigEndian.PutUint16(buf[0:2], h.ID)

	var flags uint16

	//building flags here
	if h.QR {
		flags |= maskQR
	}

	flags |= (uint16(h.Opcode) & 0xF) << 11

	if h.AA {
		flags |= maskAA
	}

	if h.TC {
		flags |= maskTC
	}

	if h.RD {
		flags |= maskRD
	}

	if h.RA {
		flags |= maskRA
	}

	flags |= uint16(h.RCode) & 0xF

	binary.BigEndian.PutUint16(buf[2:4], flags)

	binary.BigEndian.PutUint16(buf[4:6], h.QDCount)
	binary.BigEndian.PutUint16(buf[6:8], h.ANCount)
	binary.BigEndian.PutUint16(buf[8:10], h.NSCount)
	binary.BigEndian.PutUint16(buf[10:12], h.ARCount)

	return buf
}

type Message struct {
	Header      Header
	Questions   []Question
	Answers     []RR
	Authorities []RR
	Additionals []RR
}

func DecodeMessage(msg []byte) (Message, error) {
	m := Message{}

	// add header → m
	header, err := DecodeHeader(msg)
	if err != nil {
		return Message{}, err
	}
	m.Header = header

	offset := 12
	// loop and append to questions
	for i := 0; i < int(header.QDCount); i++ {
		q, newOffset, err := DecodeQuestion(msg, offset)
		if err != nil {
			return Message{}, err
		}
		offset = newOffset
		m.Questions = append(m.Questions, q)
	}

	// loop and append to answers
	for i := 0; i < int(header.ANCount); i++ {
		rr, newOffset, err := DecodeRR(msg, offset)
		if err != nil {
			return Message{}, err
		}
		offset = newOffset
		m.Answers = append(m.Answers, rr)
	}
	// loop and append to authorities
	for i := 0; i < int(header.NSCount); i++ {
		rr, newOffset, err := DecodeRR(msg, offset)
		if err != nil {
			return Message{}, err
		}
		offset = newOffset
		m.Authorities = append(m.Authorities, rr)
	}
	// loop and append to additionals
	for i := 0; i < int(header.ARCount); i++ {
		rr, newOffset, err := DecodeRR(msg, offset)
		if err != nil {
			return Message{}, err
		}
		offset = newOffset
		m.Additionals = append(m.Additionals, rr)
	}

	return m, nil
}

func (m Message) Encode() ([]byte, error) {
	m.Header.QDCount = uint16(len(m.Questions))
	m.Header.ANCount = uint16(len(m.Answers))
	m.Header.NSCount = uint16(len(m.Authorities))
	m.Header.ARCount = uint16(len(m.Additionals))

	// buf → header bytes
	buf := m.Header.Encode()

	// append questions
	for _, q := range m.Questions {
		qBytes, err := EncodeQuestion(q)
		if err != nil {
			return nil, err
		}
		buf = append(buf, qBytes...)
	}
	// append answers
	for _, rr := range m.Answers {
		aBytes, err := rr.Encode()
		if err != nil {
			return nil, err
		}
		buf = append(buf, aBytes...)
	}
	// append authorities
	for _, rr := range m.Authorities {
		authBytes, err := rr.Encode()
		if err != nil {
			return nil, err
		}
		buf = append(buf, authBytes...)
	}
	// append additionals
	for _, rr := range m.Additionals {
		addBytes, err := rr.Encode()
		if err != nil {
			return nil, err
		}
		buf = append(buf, addBytes...)
	}

	return buf, nil
}

func BuildQuery(name string, qtype uint16) ([]byte, error) {
	id := uint16(rand.Intn(65536))
	msg := Message{
		Header: Header{
			ID:      id,    // random id
			QR:      false, // query
			Opcode:  0,     // standard query
			RD:      false, // no recursion desired
			QDCount: 1,
			ANCount: 0,
			NSCount: 0,
			ARCount: 0,
		},
		Questions: []Question{
			{Name: name, Type: qtype, Class: 1},
		},
		Answers:     []RR{},
		Authorities: []RR{},
		Additionals: []RR{},
	}

	return msg.Encode()
}
