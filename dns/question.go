package dns

import (
	"encoding/binary"
	"fmt"
)

type Name = string

type Question struct {
	Name  Name
	Type  uint16
	Class uint16
}

func DecodeQuestion(msg []byte, offset int) (Question, int, error) {

	name, consumed, err := DecodeName(msg, offset)
	if err != nil {
		return Question{}, 0, err
	}
	offset += consumed

	if offset+4 > len(msg) {
		return Question{}, 0, fmt.Errorf("question too short")
	}

	q := Question{}
	q.Name = name

	q.Type = binary.BigEndian.Uint16(msg[0+offset : 2+offset])
	q.Class = binary.BigEndian.Uint16(msg[2+offset : 4+offset])

	return q, offset + 4, nil
}

func EncodeQuestion(q Question) ([]byte, error) {

	var buf []byte

	nameBuf, err := EncodeName(q.Name)
	if err != nil {
		return nil, err
	}

	buf = append(buf, nameBuf...)

	buf = append(buf, 0, 0, 0, 0)

	binary.BigEndian.PutUint16(buf[len(buf)-4:], q.Type)
	binary.BigEndian.PutUint16(buf[len(buf)-2:], q.Class)

	return buf, nil
}
