package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
)

const (
	Codec8E byte = 0x8E
	Codec12 byte = 0x0C
	Codec13 byte = 0x0D
)

var crc16Table = []uint16{
	0x0000, 0x8005, 0x800F, 0x000A, 0x801B, 0x001E, 0x0014, 0x8011,
	0x8033, 0x0036, 0x003C, 0x8039, 0x0028, 0x802D, 0x8027, 0x0022,
	0x8063, 0x0066, 0x006C, 0x8069, 0x0078, 0x807D, 0x8077, 0x0072,
	0x0050, 0x8055, 0x805F, 0x005A, 0x804B, 0x004E, 0x0044, 0x8041,
	0x80C3, 0x00C6, 0x00CC, 0x80C9, 0x00D8, 0x80DD, 0x80D7, 0x00D2,
	0x00F0, 0x80F5, 0x80FF, 0x00FA, 0x80EB, 0x00EE, 0x00E4, 0x80E1,
	0x00A0, 0x80A5, 0x80AF, 0x00AA, 0x80BB, 0x00BE, 0x00B4, 0x80B1,
	0x8093, 0x0096, 0x009C, 0x8099, 0x0088, 0x808D, 0x8087, 0x0082,
	0x8183, 0x0186, 0x018C, 0x8189, 0x0198, 0x819D, 0x8197, 0x0192,
	0x01B0, 0x81B5, 0x81BF, 0x01BA, 0x81AB, 0x01AE, 0x01A4, 0x81A1,
	0x01E0, 0x81E5, 0x81EF, 0x01EA, 0x81FB, 0x01FE, 0x01F4, 0x81F1,
	0x81D3, 0x01D6, 0x01DC, 0x81D9, 0x01C8, 0x81CD, 0x81C7, 0x01C2,
	0x0140, 0x8145, 0x814F, 0x014A, 0x815B, 0x015E, 0x0154, 0x8151,
	0x8173, 0x0176, 0x017C, 0x8179, 0x0168, 0x816D, 0x8167, 0x0162,
	0x8123, 0x0126, 0x012C, 0x8129, 0x0138, 0x813D, 0x8137, 0x0132,
	0x0110, 0x8115, 0x811F, 0x011A, 0x810B, 0x010E, 0x0104, 0x8101,
	0x8303, 0x0306, 0x030C, 0x8309, 0x0318, 0x831D, 0x8317, 0x0312,
	0x0330, 0x8335, 0x833F, 0x033A, 0x832B, 0x032E, 0x0324, 0x8321,
	0x0360, 0x8365, 0x836F, 0x036A, 0x837B, 0x037E, 0x0374, 0x8371,
	0x8353, 0x0356, 0x035C, 0x8359, 0x0348, 0x834D, 0x8347, 0x0342,
	0x03C0, 0x83C5, 0x83CF, 0x03CA, 0x83DB, 0x03DE, 0x03D4, 0x83D1,
	0x83F3, 0x03F6, 0x03FC, 0x83F9, 0x03E8, 0x83ED, 0x83E7, 0x03E2,
	0x83A3, 0x03A6, 0x03AC, 0x83A9, 0x03B8, 0x83BD, 0x83B7, 0x03B2,
	0x0390, 0x8395, 0x839F, 0x039A, 0x838B, 0x038E, 0x0384, 0x8381,
	0x0280, 0x8285, 0x828F, 0x028A, 0x829B, 0x029E, 0x0294, 0x8291,
	0x82B3, 0x02B6, 0x02BC, 0x82B9, 0x02A8, 0x82AD, 0x82A7, 0x02A2,
	0x82E3, 0x02E6, 0x02EC, 0x82E9, 0x02F8, 0x82FD, 0x82F7, 0x02F2,
	0x02D0, 0x82D5, 0x82DF, 0x02DA, 0x82CB, 0x02CE, 0x02C4, 0x82C1,
	0x8243, 0x0246, 0x024C, 0x8249, 0x0258, 0x825D, 0x8257, 0x0252,
	0x0270, 0x8275, 0x827F, 0x027A, 0x826B, 0x026E, 0x0264, 0x8261,
	0x0220, 0x8225, 0x822F, 0x022A, 0x823B, 0x023E, 0x0234, 0x8231,
	0x8213, 0x0216, 0x021C, 0x8219, 0x0208, 0x820D, 0x8207, 0x0202,
}

// crc16IBM computes the CRC-16/IBM checksum used by Codec8E packets.
func crc16IBM(data []byte) uint16 {
	var crc uint16
	for _, b := range data {
		crc = (crc << 8) ^ crc16Table[((crc>>8)^uint16(b))&0xFF]
	}
	return crc
}

type Codec8ERecord struct {
	Timestamp  uint64
	Priority   byte
	Longitude  int32
	Latitude   int32
	Altitude   int16
	Angle      int16
	Satellites byte
	Speed      uint16
	IOData     []byte
}

type Codec8EPacket struct {
	CodecID    byte
	DataCount1 byte
	Records    []Codec8ERecord
	DataCount2 byte
}

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

// Read8EPacket reads a Codec8E packet from the reader and validates its frame.
// It expects the 4-byte preamble, packet length, payload, and trailing CRC-16.
func Read8EPacket(r io.Reader) (*Codec8EPacket, error) {
	head := make([]byte, 8)
	if _, err := io.ReadFull(r, head); err != nil {
		return nil, err
	}
	if head[0] != 0x00 || head[1] != 0x00 || head[2] != 0x00 || head[3] != 0x00 {
		return nil, fmt.Errorf("invalid 8E preamble")
	}

	packetLen := binary.BigEndian.Uint32(head[4:8])
	if packetLen > 10*1024*1024 {
		return nil, fmt.Errorf("8E packet too large: %d", packetLen)
	}

	payload := make([]byte, packetLen)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}

	crcBuf := make([]byte, 2)
	if _, err := io.ReadFull(r, crcBuf); err != nil {
		return nil, err
	}

	if len(payload) < 2 {
		return nil, fmt.Errorf("payload too short")
	}

	if payload[0] != Codec8E {
		return nil, fmt.Errorf("unexpected codec ID: 0x%X", payload[0])
	}

	packet := &Codec8EPacket{
		CodecID:    payload[0],
		DataCount1: payload[1],
	}
	pos := 2

	records := make([]Codec8ERecord, 0, packet.DataCount1)
	for i := 0; i < int(packet.DataCount1); i++ {
		record, readBytes, err := parse8ERecord(payload[pos:])
		if err != nil {
			return nil, err
		}
		records = append(records, record)
		pos += readBytes
	}

	if pos >= len(payload) {
		return nil, fmt.Errorf("missing data count2")
	}
	packet.DataCount2 = payload[pos]
	pos++

	if packet.DataCount2 != packet.DataCount1 {
		return nil, fmt.Errorf("data count mismatch: %d != %d", packet.DataCount2, packet.DataCount1)
	}

	if pos != len(payload) {
		return nil, fmt.Errorf("unexpected payload length: expected %d, got %d", pos, len(payload))
	}

	crcValue := binary.BigEndian.Uint16(crcBuf)
	computedCRC := crc16IBM(payload)
	if crcValue != computedCRC {
		return nil, fmt.Errorf("crc mismatch: expected 0x%04X, got 0x%04X", computedCRC, crcValue)
	}

	packet.Records = records
	return packet, nil
}

// parse8ERecord parses a single Codec8E AVL record from the payload.
// It returns the record and the number of bytes consumed.
func parse8ERecord(data []byte) (Codec8ERecord, int, error) {
	if len(data) < 24 {
		return Codec8ERecord{}, 0, fmt.Errorf("record too short")
	}
	record := Codec8ERecord{
		Timestamp:  binary.BigEndian.Uint64(data[0:8]),
		Priority:   data[8],
		Longitude:  int32(binary.BigEndian.Uint32(data[9:13])),
		Latitude:   int32(binary.BigEndian.Uint32(data[13:17])),
		Altitude:   int16(binary.BigEndian.Uint16(data[17:19])),
		Angle:      int16(binary.BigEndian.Uint16(data[19:21])),
		Satellites: data[21],
		Speed:      binary.BigEndian.Uint16(data[22:24]),
	}
	pos := 24

	if len(data) < pos+5 {
		return Codec8ERecord{}, 0, fmt.Errorf("missing io counts")
	}
	totalIOCount := int(data[pos])
	oneByteCount := int(data[pos+1])
	twoByteCount := int(data[pos+2])
	fourByteCount := int(data[pos+3])
	eightByteCount := int(data[pos+4])
	pos += 5

	ioLen := oneByteCount*2 + twoByteCount*3 + fourByteCount*5 + eightByteCount*9
	if len(data) < pos+ioLen {
		return Codec8ERecord{}, 0, fmt.Errorf("io payload too short")
	}

	ioData := make([]byte, 5+ioLen)
	copy(ioData, data[pos-5:pos+ioLen])
	record.IOData = ioData

	readPos := pos
	for j := 0; j < oneByteCount; j++ {
		if readPos+2 > len(data) {
			return Codec8ERecord{}, 0, fmt.Errorf("io record truncated")
		}
		readPos += 2
	}
	for j := 0; j < twoByteCount; j++ {
		if readPos+3 > len(data) {
			return Codec8ERecord{}, 0, fmt.Errorf("io record truncated")
		}
		readPos += 3
	}
	for j := 0; j < fourByteCount; j++ {
		if readPos+5 > len(data) {
			return Codec8ERecord{}, 0, fmt.Errorf("io record truncated")
		}
		readPos += 5
	}
	for j := 0; j < eightByteCount; j++ {
		if readPos+9 > len(data) {
			return Codec8ERecord{}, 0, fmt.Errorf("io record truncated")
		}
		readPos += 9
	}

	if totalIOCount != oneByteCount+twoByteCount+fourByteCount+eightByteCount {
		return Codec8ERecord{}, 0, fmt.Errorf("io element count mismatch: expected %d got %d", totalIOCount, oneByteCount+twoByteCount+fourByteCount+eightByteCount)
	}

	return record, readPos, nil
}

// degreesToInt32 converts decimal degrees to Teltonika integer format.
func degreesToInt32(value float64) int32 {
	return int32(math.Round(value * 1e7))
}

// Int32ToDegrees converts Teltonika int32 coordinate format back to decimal degrees.
func Int32ToDegrees(value int32) float64 {
	return float64(value) / 1e7
}

// Build8EPacket builds a minimal Codec8E packet for the tracker to send.
// It constructs the payload from Codec ID through Number of Data 2 and appends CRC-16.
func Build8EPacket(timestamp uint64, latitude, longitude float64) ([]byte, error) {
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
	if err := binary.Write(payload, binary.BigEndian, degreesToInt32(longitude)); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.BigEndian, degreesToInt32(latitude)); err != nil {
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

	if err := payload.WriteByte(0x01); err != nil {
		return nil, err
	}
	if err := payload.WriteByte(0x01); err != nil {
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
	if err := payload.WriteByte(0x21); err != nil {
		return nil, err
	}
	if err := payload.WriteByte(0x01); err != nil {
		return nil, err
	}

	if err := payload.WriteByte(0x01); err != nil {
		return nil, err
	}

	payloadBytes := payload.Bytes()
	totalLen := uint32(len(payloadBytes))
	packet := bytes.NewBuffer(nil)
	if _, err := packet.Write([]byte{0x00, 0x00, 0x00, 0x00}); err != nil {
		return nil, err
	}
	if err := binary.Write(packet, binary.BigEndian, totalLen); err != nil {
		return nil, err
	}
	if _, err := packet.Write(payloadBytes); err != nil {
		return nil, err
	}
	crc := crc16IBM(payloadBytes)
	if err := binary.Write(packet, binary.BigEndian, crc); err != nil {
		return nil, err
	}

	return packet.Bytes(), nil
}
