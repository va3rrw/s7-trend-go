package backend

import (
	"encoding/binary"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type S7AddressSpec struct {
	Area       int // gos7 area code
	DbNumber   int
	StartByte  int
	BitNumber  int
	ByteLength int
	ValueType  TagDataType
}

const (
	s7AreaPE = 0x81
	s7AreaPA = 0x82
	s7AreaMK = 0x83
	s7AreaDB = 0x84
)

var (
	reDB   = regexp.MustCompile(`(?i)^DB(?P<db>\d+)\.(?P<kind>DBX|DBB|DBW|DBD)(?P<offset>\d+)(?:\.(?P<bit>[0-7]))?$`)
	reArea = regexp.MustCompile(`(?i)^(?:(?P<area>[MIQ])(?P<kind>B|W|D|X)?(?P<offset>\d+)(?:\.(?P<bit>[0-7]))?)$`)
)

func ParseS7Address(address string, valueType TagDataType) (*S7AddressSpec, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, fmt.Errorf("address is empty")
	}

	// Try DB pattern
	if match := reDB.FindStringSubmatch(address); match != nil {
		dbNum, _ := strconv.Atoi(match[1])
		kind := strings.ToUpper(match[2])
		offset, _ := strconv.Atoi(match[3])
		bitNum := -1
		if match[4] != "" {
			bitNum, _ = strconv.Atoi(match[4])
		}

		if kind == "DBX" && bitNum == -1 {
			return nil, fmt.Errorf("DBX addresses require a bit (e.g. DB1.DBX0.3)")
		}
		if kind != "DBX" && bitNum >= 0 {
			return nil, fmt.Errorf("only DBX addresses may include a bit number")
		}
		if bitNum >= 0 && valueType != DataTypeBool {
			return nil, fmt.Errorf("bit addresses require Bool data type")
		}
		if valueType == DataTypeBool && bitNum < 0 {
			return nil, fmt.Errorf("Bool addresses require a bit number, for example DB1.DBX0.3")
		}

		return &S7AddressSpec{
			Area:       s7AreaDB,
			DbNumber:   dbNum,
			StartByte:  offset,
			BitNumber:  bitNum,
			ByteLength: getByteLength(valueType),
			ValueType:  valueType,
		}, nil
	}

	// Try Area pattern (Memory M, Input I, Output Q)
	if match := reArea.FindStringSubmatch(address); match != nil {
		areaStr := strings.ToUpper(match[1])
		offset, _ := strconv.Atoi(match[3])
		bitNum := -1
		if match[4] != "" {
			bitNum, _ = strconv.Atoi(match[4])
		}
		kind := strings.ToUpper(match[2])
		if kind != "" && kind != "X" && bitNum >= 0 {
			return nil, fmt.Errorf("only bit addresses may include a bit number")
		}
		if bitNum >= 0 && valueType != DataTypeBool {
			return nil, fmt.Errorf("bit addresses require Bool data type")
		}
		if valueType == DataTypeBool && bitNum < 0 {
			return nil, fmt.Errorf("Bool addresses require a bit number, for example I31.1")
		}

		areaCode := s7AreaMK
		switch areaStr {
		case "I":
			areaCode = s7AreaPE
		case "Q":
			areaCode = s7AreaPA
		}

		return &S7AddressSpec{
			Area:       areaCode,
			DbNumber:   0,
			StartByte:  offset,
			BitNumber:  bitNum,
			ByteLength: getByteLength(valueType),
			ValueType:  valueType,
		}, nil
	}

	return nil, fmt.Errorf("invalid address format '%s'. Use DB1.DBD0, MB0, IB0, QB0 or M0.0", address)
}

func getByteLength(valueType TagDataType) int {
	switch valueType {
	case DataTypeBool, DataTypeByte:
		return 1
	case DataTypeWord, DataTypeInt:
		return 2
	case DataTypeDWord, DataTypeDInt, DataTypeReal:
		return 4
	case DataTypeLReal:
		return 8
	default:
		return 4
	}
}

func DecodeS7Value(data []byte, valueType TagDataType, bitNumber int) (string, *float64) {
	if len(data) == 0 {
		return "-", nil
	}

	switch valueType {
	case DataTypeBool:
		bit := 0
		if bitNumber >= 0 && bitNumber <= 7 {
			bit = bitNumber
		}
		val := (data[0] & (1 << bit)) != 0
		num := 0.0
		if val {
			num = 1.0
		}
		return fmt.Sprintf("%t", val), &num

	case DataTypeByte:
		val := data[0]
		num := float64(val)
		return fmt.Sprintf("%d", val), &num

	case DataTypeWord:
		if len(data) < 2 {
			return "-", nil
		}
		val := binary.BigEndian.Uint16(data)
		num := float64(val)
		return fmt.Sprintf("%d", val), &num

	case DataTypeInt:
		if len(data) < 2 {
			return "-", nil
		}
		val := int16(binary.BigEndian.Uint16(data))
		num := float64(val)
		return fmt.Sprintf("%d", val), &num

	case DataTypeDWord:
		if len(data) < 4 {
			return "-", nil
		}
		val := binary.BigEndian.Uint32(data)
		num := float64(val)
		return fmt.Sprintf("%d", val), &num

	case DataTypeDInt:
		if len(data) < 4 {
			return "-", nil
		}
		val := int32(binary.BigEndian.Uint32(data))
		num := float64(val)
		return fmt.Sprintf("%d", val), &num

	case DataTypeReal:
		if len(data) < 4 {
			return "-", nil
		}
		bits := binary.BigEndian.Uint32(data)
		val := math.Float32frombits(bits)
		num := float64(val)
		return fmt.Sprintf("%.3f", val), &num

	case DataTypeLReal:
		if len(data) < 8 {
			return "-", nil
		}
		bits := binary.BigEndian.Uint64(data)
		val := math.Float64frombits(bits)
		return fmt.Sprintf("%.3f", val), &val
	}

	return "-", nil
}
