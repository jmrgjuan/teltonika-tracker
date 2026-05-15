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

func ReadIMEI(r io.Reader) (string, error) {
	lengthBuf := make([]byte, 2)
	if _, err := io.ReadFull(r, lengthBuf); err != nil {
		return "", err
	}
	imeiLen := binary.BigEndian.Uint16(lengthBuf)
	if imeiLen == 0 || imeiLen > 32 {
		return "", fmt.Errorf("invalid IMEI length: %d", imeiLen)
	}
	imeiBuf := make([]byte, imeiLen)
	if _, err := io.ReadFull(r, imeiBuf); err != nil {
		return "", err
	}
	return string(imeiBuf), nil
}

func SendIMEIResponse(conn net.Conn, ok bool) error {
	ack := byte(0x00)
	if ok {
		ack = 0x01
	}
	_, err := conn.Write([]byte{ack})
	return err
}

func Read8EPacket(r io.Reader) ([]byte, error) {
	header := make([]byte, 6)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}

	payloadLen := binary.BigEndian.Uint32(header[2:6])
	if payloadLen > 10*1024*1024 {
		return nil, fmt.Errorf("8E packet too large: %d", payloadLen)
	}

	payload := make([]byte, payloadLen)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}

	crcBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, crcBuf); err != nil {
		return nil, err
	}
	crc := binary.BigEndian.Uint32(crcBuf)
	if crc != crc32.ChecksumIEEE(payload) {
		return nil, fmt.Errorf("crc mismatch: expected 0x%08X, got 0x%08X", crc32.ChecksumIEEE(payload), crc)
	}
	if len(payload) == 0 || payload[0] != Codec8E {
		return nil, fmt.Errorf("unexpected codec byte: 0x%X", payload[0])
	}

	return payload, nil
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
