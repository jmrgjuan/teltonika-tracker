package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"net"
)

const (
	Codec8E byte = 0x8E
	Codec12 byte = 0x0C
	Codec13 byte = 0x0D
)

func ParseCodec(value string) (byte, error) {
	switch value {
	case "8e", "8E":
		return Codec8E, nil
	case "12":
		return Codec12, nil
	case "13":
		return Codec13, nil
	default:
		return 0, fmt.Errorf("unsupported codec: %q, use 8e, 12, or 13", value)
	}
}

func SendIMEI(conn net.Conn, imei string) error {
	if len(imei) < 5 || len(imei) > 15 {
		return fmt.Errorf("imei must be between 5 and 15 digits")
	}

	buf := bytes.NewBuffer(nil)
	if err := binary.Write(buf, binary.BigEndian, uint16(len(imei))); err != nil {
		return err
	}
	if _, err := buf.WriteString(imei); err != nil {
		return err
	}

	_, err := conn.Write(buf.Bytes())
	return err
}

func ReadIMEIResponse(r io.Reader) (bool, error) {
	ack := make([]byte, 1)
	if _, err := io.ReadFull(r, ack); err != nil {
		return false, err
	}
	return ack[0] == 0x01, nil
}

func BuildLoginPacket(imei string) ([]byte, error) {
	if len(imei) < 5 || len(imei) > 15 {
		return nil, fmt.Errorf("imei must be between 5 and 15 digits")
	}
	buf := bytes.NewBuffer(nil)
	if err := binary.Write(buf, binary.BigEndian, uint16(len(imei))); err != nil {
		return nil, err
	}
	if _, err := buf.WriteString(imei); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Build8EPacket(timestamp uint64) ([]byte, error) {
	payload := bytes.NewBuffer(nil)

	if err := payload.WriteByte(Codec8E); err != nil {
		return nil, err
	}
	if err := payload.WriteByte(0x01); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.BigEndian, timestamp); err != nil {
		return nil, err
	}
	if err := payload.WriteByte(0x00); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.BigEndian, int32(0)); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.BigEndian, int32(0)); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.BigEndian, int16(0)); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.BigEndian, int16(0)); err != nil {
		return nil, err
	}
	if err := payload.WriteByte(0x00); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.BigEndian, uint16(0)); err != nil {
		return nil, err
	}

	// IO element counts: total elements plus each size category.
	if err := payload.WriteByte(0x00); err != nil {
		return nil, err
	}
	if err := payload.WriteByte(0x00); err != nil {
		return nil, err
	}
	if err := payload.WriteByte(0x00); err != nil {
		return nil, err
	}
	if err := payload.WriteByte(0x00); err != nil {
		return nil, err
	}
	if err := payload.WriteByte(0x01); err != nil {
		return nil, err
	}

	payloadBytes := payload.Bytes()
	packet := bytes.NewBuffer(nil)
	if err := binary.Write(packet, binary.BigEndian, uint16(0)); err != nil {
		return nil, err
	}
	if err := binary.Write(packet, binary.BigEndian, uint32(len(payloadBytes))); err != nil {
		return nil, err
	}
	if _, err := packet.Write(payloadBytes); err != nil {
		return nil, err
	}
	crc := crc32.ChecksumIEEE(payloadBytes)
	if err := binary.Write(packet, binary.BigEndian, crc); err != nil {
		return nil, err
	}

	return packet.Bytes(), nil
}
