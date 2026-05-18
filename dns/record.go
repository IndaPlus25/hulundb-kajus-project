package dns

import (
	"encoding/binary"
	"fmt"
	"net"
)

const (
	TypeA     uint16 = 1
	TypeNS    uint16 = 2
	TypeCNAME uint16 = 5
	TypeSOA   uint16 = 6
	TypeMX    uint16 = 15
	TypeAAAA  uint16 = 28
)

type RData interface {
	Encode() ([]byte, error)
}

type ARecord struct {
	IP net.IP
}

type AAAARecord struct {
	IP net.IP
}

type NSRecord struct {
	Name string
}

type CNAMERecord struct {
	Name string
}

type SOARecord struct {
	MName   string
	RName   string
	Serial  uint32
	Refresh uint32
	Retry   uint32
	Expire  uint32
	Minimum uint32
}

type MXRecord struct {
	Preference uint16
	Exchange   string
}

func (r ARecord) Encode() ([]byte, error) {
	ip := r.IP.To4()
	if ip == nil {
		return nil, fmt.Errorf("not a valid IPv4 address")
	}
	return ip, nil
}

func (r AAAARecord) Encode() ([]byte, error) {
	ip := r.IP.To16()
	if ip == nil {
		return nil, fmt.Errorf("not a valid IPv6 address")
	}
	return ip, nil
}

func (r NSRecord) Encode() ([]byte, error) {
	return EncodeName(r.Name)
}

func (r CNAMERecord) Encode() ([]byte, error) {
	return EncodeName(r.Name)
}

func (r SOARecord) Encode() ([]byte, error) {
	var buf []byte

	mname, err := EncodeName(r.MName)
	if err != nil {
		return nil, err // Return early if the first name is invalid
	}

	rname, err := EncodeName(r.RName)
	if err != nil {
		return nil, err
	}

	buf = append(buf, mname...)
	buf = append(buf, rname...)

	timers := make([]byte, 20)
	binary.BigEndian.PutUint32(timers[0:4], r.Serial)
	binary.BigEndian.PutUint32(timers[4:8], r.Refresh)
	binary.BigEndian.PutUint32(timers[8:12], r.Retry)
	binary.BigEndian.PutUint32(timers[12:16], r.Expire)
	binary.BigEndian.PutUint32(timers[16:20], r.Minimum)

	buf = append(buf, timers...)

	return buf, nil
}

func (r MXRecord) Encode() ([]byte, error) {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf[0:2], r.Preference)

	exchange, err := EncodeName(r.Exchange)
	if err != nil {
		return nil, err
	}

	buf = append(buf, exchange...)
	return buf, nil
}

func DecodeARecord(rdata []byte) (ARecord, error) {
	if len(rdata) != 4 {
		return ARecord{}, fmt.Errorf("not a valid IPv4 address")
	}
	return ARecord{IP: net.IP(rdata)}, nil
}

func DecodeAAAARecord(rdata []byte) (AAAARecord, error) {
	if len(rdata) != 16 {
		return AAAARecord{}, fmt.Errorf("not a valid IPv6 address")
	}
	return AAAARecord{IP: net.IP(rdata)}, nil
}

func DecodeNSRecord(msg []byte, offset int) (NSRecord, error) {
	name, _, err := DecodeName(msg, offset)
	if err != nil {
		return NSRecord{}, err
	}

	return NSRecord{Name: name}, nil
}

func DecodeCNAMERecord(msg []byte, offset int) (CNAMERecord, error) {
	name, _, err := DecodeName(msg, offset)
	if err != nil {
		return CNAMERecord{}, err
	}

	return CNAMERecord{Name: name}, nil
}

func DecodeMXRecord(msg []byte, offset int) (MXRecord, error) {
	if offset+2 > len(msg) {
		return MXRecord{}, fmt.Errorf("not enough bytes")
	}

	preferance := binary.BigEndian.Uint16(msg[offset : offset+2])
	name, _, err := DecodeName(msg, offset+2)
	if err != nil {
		return MXRecord{}, err
	}

	return MXRecord{Preference: preferance, Exchange: name}, nil

}

func DecodeSOARecord(msg []byte, offset int) (SOARecord, error) {
	r := SOARecord{}

	mname, consumed, err := DecodeName(msg, offset)
	if err != nil {
		return SOARecord{}, err
	}
	offset += consumed

	rname, consumed, err := DecodeName(msg, offset)
	if err != nil {
		return SOARecord{}, err
	}
	offset += consumed

	if offset+20 > len(msg) {
		return SOARecord{}, fmt.Errorf("not enough bytes")
	}

	r.MName = mname
	r.RName = rname

	r.Serial = binary.BigEndian.Uint32(msg[0+offset : 4+offset])
	r.Refresh = binary.BigEndian.Uint32(msg[4+offset : 8+offset])
	r.Retry = binary.BigEndian.Uint32(msg[8+offset : 12+offset])
	r.Expire = binary.BigEndian.Uint32(msg[12+offset : 16+offset])
	r.Minimum = binary.BigEndian.Uint32(msg[16+offset : 20+offset])

	return r, nil

}

func DecodeRData(msg []byte, offset int, rrType uint16, rdLength uint16) (RData, error) {

	if offset+int(rdLength) > len(msg) {
		return nil, fmt.Errorf("rdata out of bounds")
	}
	rdata := msg[offset : offset+int(rdLength)]

	switch rrType {
	case TypeA:
		return DecodeARecord(rdata)
	case TypeNS:
		return DecodeNSRecord(msg, offset)
	case TypeCNAME:
		return DecodeCNAMERecord(msg, offset)
	case TypeSOA:
		return DecodeSOARecord(msg, offset)
	case TypeMX:
		return DecodeMXRecord(msg, offset)
	case TypeAAAA:
		return DecodeAAAARecord(rdata)
	default:
		// return nil, fmt.Errorf("unknown RData type")
		return nil, nil
	}
}

type RRHeader struct {
	Name     string
	Type     uint16
	Class    uint16
	TTL      uint32
	RDLength uint16
}

func (h RRHeader) Encode() ([]byte, error) {
	name, err := EncodeName(h.Name)
	if err != nil {
		return nil, err
	}

	parts := make([]byte, 10)
	binary.BigEndian.PutUint16(parts[0:2], h.Type)
	binary.BigEndian.PutUint16(parts[2:4], h.Class)
	binary.BigEndian.PutUint32(parts[4:8], h.TTL)
	binary.BigEndian.PutUint16(parts[8:10], h.RDLength)

	var buf []byte
	buf = append(buf, name...)
	buf = append(buf, parts...)

	return buf, nil
}

func DecodeRRHeader(msg []byte, offset int) (RRHeader, int, error) {
	h := RRHeader{}

	name, consumed, err := DecodeName(msg, offset)
	if err != nil {
		return h, 0, err
	}
	offset += consumed

	if offset+10 > len(msg) {
		return RRHeader{}, 0, fmt.Errorf("not enough bytes")
	}
	h.Name = name

	h.Type = binary.BigEndian.Uint16(msg[offset : 2+offset])
	h.Class = binary.BigEndian.Uint16(msg[2+offset : 4+offset])
	h.TTL = binary.BigEndian.Uint32(msg[4+offset : 8+offset])
	h.RDLength = binary.BigEndian.Uint16(msg[8+offset : 10+offset])

	return h, offset + 10, nil
}

type RR struct {
	Header RRHeader
	Data   RData
}

func (rr RR) Encode() ([]byte, error) {
	rdata, err := rr.Data.Encode()
	if err != nil {
		return nil, err
	}

	rr.Header.RDLength = uint16(len(rdata))

	header, err := rr.Header.Encode()
	if err != nil {
		return nil, err
	}

	var buf []byte
	buf = append(buf, header...)
	buf = append(buf, rdata...)

	return buf, nil
}

func DecodeRR(msg []byte, offset int) (RR, int, error) {
	h, offset, err := DecodeRRHeader(msg, offset)
	if err != nil {
		return RR{}, 0, err
	}
	rdata, err := DecodeRData(msg, offset, h.Type, h.RDLength)
	if err != nil {
		return RR{}, 0, err
	}

	return RR{h, rdata}, offset + int(h.RDLength), nil
}
