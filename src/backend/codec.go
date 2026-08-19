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
	WordLen    int // gos7 word length (0x02 for Byte, 0x1C for Counter, 0x1D for Timer)
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
	s7AreaCT = 0x1C // Counter
	s7AreaTM = 0x1D // Timer

	s7WLTimer   = 0x1D
	s7WLCounter = 0x1C
	s7WLByte    = 0x02
)

var (
	reDB      = regexp.MustCompile(`(?i)^DB(?P<db>\d+)\.(?P<kind>DBX|DBB|DBW|DBD)(?P<offset>\d+)(?:\.(?P<bit>[0-7]))?$`)
	reArea    = regexp.MustCompile(`(?i)^(?:(?P<area>[MIQ])(?P<kind>B|W|D|X)?(?P<offset>\d+)(?:\.(?P<bit>[0-7]))?)$`)
	reTimer   = regexp.MustCompile(`(?i)^(?:T|TM)(?P<num>\d+)$`)
	reCounter = regexp.MustCompile(`(?i)^(?:C|Z|CT)(?P<num>\d+)$`)
)

func ParseS7Address(address string, valueType TagDataType) (*S7AddressSpec, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, fmt.Errorf("address is empty")
	}

	// Try Timer pattern (T0..T65535, TM0..TM65535)
	if match := reTimer.FindStringSubmatch(address); match != nil {
		num, _ := strconv.Atoi(match[1])
		return &S7AddressSpec{
			Area:       s7AreaTM,
			WordLen:    s7WLTimer,
			DbNumber:   0,
			StartByte:  num,
			BitNumber:  -1,
			ByteLength: 2,
			ValueType:  valueType,
		}, nil
	}

	// Try Counter pattern (C0..C65535, Z0..Z65535, CT0..CT65535)
	if match := reCounter.FindStringSubmatch(address); match != nil {
		num, _ := strconv.Atoi(match[1])
		return &S7AddressSpec{
			Area:       s7AreaCT,
			WordLen:    s7WLCounter,
			DbNumber:   0,
			StartByte:  num,
			BitNumber:  -1,
			ByteLength: 2,
			ValueType:  valueType,
		}, nil
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
			WordLen:    s7WLByte,
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
			WordLen:    s7WLByte,
			DbNumber:   0,
			StartByte:  offset,
			BitNumber:  bitNum,
			ByteLength: getByteLength(valueType),
			ValueType:  valueType,
		}, nil
	}

	return nil, fmt.Errorf("invalid address format '%s'. Use DB1.DBD0, MB0, IB0, QB0, T0 or C0", address)
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

// DecodeS7Timer decodes Siemens S5TIME / Timer (16-bit) into seconds
func DecodeS7Timer(data []byte) (string, *float64) {
	if len(data) < 2 {
		return "-", nil
	}
	digit0 := float64(data[1] & 0x0F)
	digit1 := float64((data[1] >> 4) & 0x0F)
	digit2 := float64(data[0] & 0x0F)
	bcd := digit2*100 + digit1*10 + digit0

	base := (data[0] >> 4) & 0x03
	var mult float64
	switch base {
	case 0:
		mult = 0.01 // 10ms
	case 1:
		mult = 0.1 // 100ms
	case 2:
		mult = 1.0 // 1s
	case 3:
		mult = 10.0 // 10s
	}
	sec := bcd * mult
	return fmt.Sprintf("%.3f", sec), &sec
}

// DecodeS7Counter decodes Siemens S7 Counter (16-bit BCD 0..999)
func DecodeS7Counter(data []byte) (string, *float64) {
	if len(data) < 2 {
		return "-", nil
	}
	digit0 := int(data[1] & 0x0F)
	digit1 := int((data[1] >> 4) & 0x0F)
	digit2 := int(data[0] & 0x0F)
	val := digit2*100 + digit1*10 + digit0
	num := float64(val)
	return fmt.Sprintf("%d", val), &num
}
