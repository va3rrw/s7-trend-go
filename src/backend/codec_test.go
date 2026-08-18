package backend

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestParseS7Address_Valid(t *testing.T) {
	tests := []struct {
		name         string
		address      string
		valueType    TagDataType
		stringLength int
		expectedArea int
		expectedDb   int
		expectedByte int
		expectedBit  int
		expectedLen  int
	}{
		// DB addresses
		{
			name:         "DB Bool bit 0",
			address:      "DB1.DBX0.0",
			valueType:    DataTypeBool,
			expectedArea: s7AreaDB,
			expectedDb:   1,
			expectedByte: 0,
			expectedBit:  0,
			expectedLen:  1,
		},
		{
			name:         "DB Bool bit 7 case insensitive",
			address:      "db10.dbx5.7",
			valueType:    DataTypeBool,
			expectedArea: s7AreaDB,
			expectedDb:   10,
			expectedByte: 5,
			expectedBit:  7,
			expectedLen:  1,
		},
		{
			name:         "DB Byte",
			address:      "DB2.DBB10",
			valueType:    DataTypeByte,
			expectedArea: s7AreaDB,
			expectedDb:   2,
			expectedByte: 10,
			expectedBit:  -1,
			expectedLen:  1,
		},
		{
			name:         "DB Word",
			address:      "DB3.DBW20",
			valueType:    DataTypeWord,
			expectedArea: s7AreaDB,
			expectedDb:   3,
			expectedByte: 20,
			expectedBit:  -1,
			expectedLen:  2,
		},
		{
			name:         "DB Int",
			address:      "DB4.DBW30",
			valueType:    DataTypeInt,
			expectedArea: s7AreaDB,
			expectedDb:   4,
			expectedByte: 30,
			expectedBit:  -1,
			expectedLen:  2,
		},
		{
			name:         "DB DWord",
			address:      "DB5.DBD40",
			valueType:    DataTypeDWord,
			expectedArea: s7AreaDB,
			expectedDb:   5,
			expectedByte: 40,
			expectedBit:  -1,
			expectedLen:  4,
		},
		{
			name:         "DB DInt",
			address:      "DB6.DBD50",
			valueType:    DataTypeDInt,
			expectedArea: s7AreaDB,
			expectedDb:   6,
			expectedByte: 50,
			expectedBit:  -1,
			expectedLen:  4,
		},
		{
			name:         "DB Real",
			address:      "DB7.DBD60",
			valueType:    DataTypeReal,
			expectedArea: s7AreaDB,
			expectedDb:   7,
			expectedByte: 60,
			expectedBit:  -1,
			expectedLen:  4,
		},
		{
			name:         "DB LReal",
			address:      "DB8.DBD70",
			valueType:    DataTypeLReal,
			expectedArea: s7AreaDB,
			expectedDb:   8,
			expectedByte: 70,
			expectedBit:  -1,
			expectedLen:  8,
		},
		{
			name:         "DB String with length 50",
			address:      "DB9.DBD80",
			valueType:    DataTypeString,
			stringLength: 50,
			expectedArea: s7AreaDB,
			expectedDb:   9,
			expectedByte: 80,
			expectedBit:  -1,
			expectedLen:  52, // 50 + 2 header bytes
		},
		{
			name:         "DB String default length when 0",
			address:      "DB9.DBD80",
			valueType:    DataTypeString,
			stringLength: 0,
			expectedArea: s7AreaDB,
			expectedDb:   9,
			expectedByte: 80,
			expectedBit:  -1,
			expectedLen:  22, // 20 + 2 header bytes
		},
		{
			name:         "DB String capped length when > 254",
			address:      "DB9.DBD80",
			valueType:    DataTypeString,
			stringLength: 300,
			expectedArea: s7AreaDB,
			expectedDb:   9,
			expectedByte: 80,
			expectedBit:  -1,
			expectedLen:  256, // 254 + 2 header bytes
		},

		// Memory (M) area
		{
			name:         "Memory Bool bit M0.0",
			address:      "M0.0",
			valueType:    DataTypeBool,
			expectedArea: s7AreaMK,
			expectedDb:   0,
			expectedByte: 0,
			expectedBit:  0,
			expectedLen:  1,
		},
		{
			name:         "Memory Bool bit MX2.4",
			address:      "MX2.4",
			valueType:    DataTypeBool,
			expectedArea: s7AreaMK,
			expectedDb:   0,
			expectedByte: 2,
			expectedBit:  4,
			expectedLen:  1,
		},
		{
			name:         "Memory Byte MB10",
			address:      "MB10",
			valueType:    DataTypeByte,
			expectedArea: s7AreaMK,
			expectedDb:   0,
			expectedByte: 10,
			expectedBit:  -1,
			expectedLen:  1,
		},
		{
			name:         "Memory Word MW20",
			address:      "MW20",
			valueType:    DataTypeWord,
			expectedArea: s7AreaMK,
			expectedDb:   0,
			expectedByte: 20,
			expectedBit:  -1,
			expectedLen:  2,
		},
		{
			name:         "Memory DWord MD30",
			address:      "MD30",
			valueType:    DataTypeDWord,
			expectedArea: s7AreaMK,
			expectedDb:   0,
			expectedByte: 30,
			expectedBit:  -1,
			expectedLen:  4,
		},

		// Input (I) area
		{
			name:         "Input Bool bit I0.3",
			address:      "I0.3",
			valueType:    DataTypeBool,
			expectedArea: s7AreaPE,
			expectedDb:   0,
			expectedByte: 0,
			expectedBit:  3,
			expectedLen:  1,
		},
		{
			name:         "Input Byte IB4",
			address:      "IB4",
			valueType:    DataTypeByte,
			expectedArea: s7AreaPE,
			expectedDb:   0,
			expectedByte: 4,
			expectedBit:  -1,
			expectedLen:  1,
		},
		{
			name:         "Input Word IW8",
			address:      "IW8",
			valueType:    DataTypeInt,
			expectedArea: s7AreaPE,
			expectedDb:   0,
			expectedByte: 8,
			expectedBit:  -1,
			expectedLen:  2,
		},
		{
			name:         "Input DWord ID12",
			address:      "ID12",
			valueType:    DataTypeReal,
			expectedArea: s7AreaPE,
			expectedDb:   0,
			expectedByte: 12,
			expectedBit:  -1,
			expectedLen:  4,
		},

		// Output (Q) area
		{
			name:         "Output Bool bit Q1.5",
			address:      "Q1.5",
			valueType:    DataTypeBool,
			expectedArea: s7AreaPA,
			expectedDb:   0,
			expectedByte: 1,
			expectedBit:  5,
			expectedLen:  1,
		},
		{
			name:         "Output Byte QB6",
			address:      "QB6",
			valueType:    DataTypeByte,
			expectedArea: s7AreaPA,
			expectedDb:   0,
			expectedByte: 6,
			expectedBit:  -1,
			expectedLen:  1,
		},
		{
			name:         "Output Word QW16",
			address:      "QW16",
			valueType:    DataTypeWord,
			expectedArea: s7AreaPA,
			expectedDb:   0,
			expectedByte: 16,
			expectedBit:  -1,
			expectedLen:  2,
		},
		{
			name:         "Output DWord QD24",
			address:      "QD24",
			valueType:    DataTypeDInt,
			expectedArea: s7AreaPA,
			expectedDb:   0,
			expectedByte: 24,
			expectedBit:  -1,
			expectedLen:  4,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := ParseS7Address(tc.address, tc.valueType, tc.stringLength)
			if err != nil {
				t.Fatalf("unexpected error parsing '%s': %v", tc.address, err)
			}
			if spec.Area != tc.expectedArea {
				t.Errorf("expected area 0x%X, got 0x%X", tc.expectedArea, spec.Area)
			}
			if spec.DbNumber != tc.expectedDb {
				t.Errorf("expected DbNumber %d, got %d", tc.expectedDb, spec.DbNumber)
			}
			if spec.StartByte != tc.expectedByte {
				t.Errorf("expected StartByte %d, got %d", tc.expectedByte, spec.StartByte)
			}
			if spec.BitNumber != tc.expectedBit {
				t.Errorf("expected BitNumber %d, got %d", tc.expectedBit, spec.BitNumber)
			}
			if spec.ByteLength != tc.expectedLen {
				t.Errorf("expected ByteLength %d, got %d", tc.expectedLen, spec.ByteLength)
			}
			if spec.ValueType != tc.valueType {
				t.Errorf("expected ValueType %s, got %s", tc.valueType, spec.ValueType)
			}
		})
	}
}

func TestParseS7Address_Invalid(t *testing.T) {
	tests := []struct {
		name         string
		address      string
		valueType    TagDataType
		stringLength int
		errContains  string
	}{
		{
			name:        "Empty address",
			address:     "   ",
			valueType:   DataTypeBool,
			errContains: "address is empty",
		},
		{
			name:        "DBX without bit number",
			address:     "DB1.DBX0",
			valueType:   DataTypeBool,
			errContains: "DBX addresses require a bit",
		},
		{
			name:        "DBW with bit number",
			address:     "DB1.DBW0.3",
			valueType:   DataTypeWord,
			errContains: "only DBX addresses may include a bit number",
		},
		{
			name:        "DBX bit with non-Bool type",
			address:     "DB1.DBX0.3",
			valueType:   DataTypeInt,
			errContains: "bit addresses require Bool data type",
		},
		{
			name:        "DBB with Bool type (missing bit number)",
			address:     "DB1.DBB0",
			valueType:   DataTypeBool,
			errContains: "Bool addresses require a bit number",
		},
		{
			name:        "MB with bit number",
			address:     "MB0.2",
			valueType:   DataTypeBool,
			errContains: "only bit addresses may include a bit number",
		},
		{
			name:        "Memory bit with non-Bool type",
			address:     "M0.2",
			valueType:   DataTypeInt,
			errContains: "bit addresses require Bool data type",
		},
		{
			name:        "Memory Byte with Bool type (missing bit)",
			address:     "MB0",
			valueType:   DataTypeBool,
			errContains: "Bool addresses require a bit number",
		},
		{
			name:        "Completely invalid format",
			address:     "INVALID_FORMAT_123",
			valueType:   DataTypeReal,
			errContains: "invalid address format",
		},
		{
			name:        "Invalid area letter",
			address:     "XB10",
			valueType:   DataTypeByte,
			errContains: "invalid address format",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := ParseS7Address(tc.address, tc.valueType, tc.stringLength)
			if err == nil {
				t.Fatalf("expected error for address '%s', but got spec %+v", tc.address, spec)
			}
			if tc.errContains != "" && !containsSubstr(err.Error(), tc.errContains) {
				t.Errorf("expected error to contain '%s', got '%s'", tc.errContains, err.Error())
			}
		})
	}
}

func containsSubstr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && func() bool {
		for i := 0; i+len(substr) <= len(s); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}()))
}

func TestGetByteLength(t *testing.T) {
	tests := []struct {
		valType  TagDataType
		strLen   int
		expected int
	}{
		{DataTypeBool, 0, 1},
		{DataTypeByte, 0, 1},
		{DataTypeWord, 0, 2},
		{DataTypeInt, 0, 2},
		{DataTypeDWord, 0, 4},
		{DataTypeDInt, 0, 4},
		{DataTypeReal, 0, 4},
		{DataTypeLReal, 0, 8},
		{DataTypeString, 0, 22},   // default 20 + 2
		{DataTypeString, -5, 22},  // min clamp to 20 + 2
		{DataTypeString, 40, 42},  // 40 + 2
		{DataTypeString, 300, 256},// max clamp to 254 + 2
		{TagDataType("Unknown"), 0, 4}, // default fallback
	}

	for _, tc := range tests {
		t.Run(string(tc.valType), func(t *testing.T) {
			got := getByteLength(tc.valType, tc.strLen)
			if got != tc.expected {
				t.Errorf("getByteLength(%s, %d) = %d; expected %d", tc.valType, tc.strLen, got, tc.expected)
			}
		})
	}
}

func TestDecodeS7Value(t *testing.T) {
	t.Run("Empty data", func(t *testing.T) {
		str, num := DecodeS7Value([]byte{}, DataTypeInt, -1)
		if str != "-" || num != nil {
			t.Errorf("expected '-', nil for empty data, got '%s', %v", str, num)
		}
	})

	t.Run("DataTypeBool", func(t *testing.T) {
		// Bit 0 true
		str, num := DecodeS7Value([]byte{0x01}, DataTypeBool, 0)
		if str != "true" || num == nil || *num != 1.0 {
			t.Errorf("expected 'true', 1.0; got '%s', %v", str, num)
		}

		// Bit 0 false
		str, num = DecodeS7Value([]byte{0xFE}, DataTypeBool, 0)
		if str != "false" || num == nil || *num != 0.0 {
			t.Errorf("expected 'false', 0.0; got '%s', %v", str, num)
		}

		// Bit 3 true
		str, num = DecodeS7Value([]byte{0x08}, DataTypeBool, 3)
		if str != "true" || num == nil || *num != 1.0 {
			t.Errorf("expected 'true', 1.0; got '%s', %v", str, num)
		}

		// Bit 3 false
		str, num = DecodeS7Value([]byte{0xF7}, DataTypeBool, 3)
		if str != "false" || num == nil || *num != 0.0 {
			t.Errorf("expected 'false', 0.0; got '%s', %v", str, num)
		}

		// Invalid bit index defaults to bit 0
		str, num = DecodeS7Value([]byte{0x01}, DataTypeBool, -1)
		if str != "true" || num == nil || *num != 1.0 {
			t.Errorf("expected 'true', 1.0; got '%s', %v", str, num)
		}
	})

	t.Run("DataTypeByte", func(t *testing.T) {
		str, num := DecodeS7Value([]byte{0x7F}, DataTypeByte, -1)
		if str != "127" || num == nil || *num != 127.0 {
			t.Errorf("expected '127', 127.0; got '%s', %v", str, num)
		}
	})

	t.Run("DataTypeWord", func(t *testing.T) {
		// Valid 2 bytes
		str, num := DecodeS7Value([]byte{0x01, 0x00}, DataTypeWord, -1)
		if str != "256" || num == nil || *num != 256.0 {
			t.Errorf("expected '256', 256.0; got '%s', %v", str, num)
		}

		// Truncated data
		str, num = DecodeS7Value([]byte{0x01}, DataTypeWord, -1)
		if str != "-" || num != nil {
			t.Errorf("expected '-', nil for truncated word, got '%s', %v", str, num)
		}
	})

	t.Run("DataTypeInt", func(t *testing.T) {
		// Negative int16: -2 (0xFFFE)
		data := make([]byte, 2)
		binary.BigEndian.PutUint16(data, 0xFFFE)
		str, num := DecodeS7Value(data, DataTypeInt, -1)
		if str != "-2" || num == nil || *num != -2.0 {
			t.Errorf("expected '-2', -2.0; got '%s', %v", str, num)
		}

		// Positive int16: 32767
		binary.BigEndian.PutUint16(data, 32767)
		str, num = DecodeS7Value(data, DataTypeInt, -1)
		if str != "32767" || num == nil || *num != 32767.0 {
			t.Errorf("expected '32767', 32767.0; got '%s', %v", str, num)
		}

		// Truncated data
		str, num = DecodeS7Value([]byte{0x01}, DataTypeInt, -1)
		if str != "-" || num != nil {
			t.Errorf("expected '-', nil for truncated int, got '%s', %v", str, num)
		}
	})

	t.Run("DataTypeDWord", func(t *testing.T) {
		data := make([]byte, 4)
		binary.BigEndian.PutUint32(data, 65536)
		str, num := DecodeS7Value(data, DataTypeDWord, -1)
		if str != "65536" || num == nil || *num != 65536.0 {
			t.Errorf("expected '65536', 65536.0; got '%s', %v", str, num)
		}

		// Truncated
		str, num = DecodeS7Value([]byte{0x00, 0x01}, DataTypeDWord, -1)
		if str != "-" || num != nil {
			t.Errorf("expected '-', nil for truncated dword, got '%s', %v", str, num)
		}
	})

	t.Run("DataTypeDInt", func(t *testing.T) {
		data := make([]byte, 4)
		// -100000 in two's complement 32-bit: 0xFFFE7960
		binary.BigEndian.PutUint32(data, 0xFFFE7960)
		str, num := DecodeS7Value(data, DataTypeDInt, -1)
		if str != "-100000" || num == nil || *num != -100000.0 {
			t.Errorf("expected '-100000', -100000.0; got '%s', %v", str, num)
		}

		// Truncated
		str, num = DecodeS7Value([]byte{0x00, 0x01}, DataTypeDInt, -1)
		if str != "-" || num != nil {
			t.Errorf("expected '-', nil for truncated dint, got '%s', %v", str, num)
		}
	})

	t.Run("DataTypeReal", func(t *testing.T) {
		data := make([]byte, 4)
		binary.BigEndian.PutUint32(data, math.Float32bits(123.456))
		str, num := DecodeS7Value(data, DataTypeReal, -1)
		if str != "123.456" || num == nil {
			t.Errorf("expected '123.456', got '%s', %v", str, num)
		}
		if math.Abs(*num-123.456) > 0.001 {
			t.Errorf("expected numeric ~123.456, got %f", *num)
		}

		// Truncated
		str, num = DecodeS7Value([]byte{0x00, 0x01}, DataTypeReal, -1)
		if str != "-" || num != nil {
			t.Errorf("expected '-', nil for truncated real, got '%s', %v", str, num)
		}
	})

	t.Run("DataTypeLReal", func(t *testing.T) {
		data := make([]byte, 8)
		binary.BigEndian.PutUint64(data, math.Float64bits(987654.321))
		str, num := DecodeS7Value(data, DataTypeLReal, -1)
		if str != "987654.321" || num == nil {
			t.Errorf("expected '987654.321', got '%s', %v", str, num)
		}
		if math.Abs(*num-987654.321) > 0.0001 {
			t.Errorf("expected numeric ~987654.321, got %f", *num)
		}

		// Truncated
		str, num = DecodeS7Value([]byte{0x00, 0x01, 0x02, 0x03}, DataTypeLReal, -1)
		if str != "-" || num != nil {
			t.Errorf("expected '-', nil for truncated lreal, got '%s', %v", str, num)
		}
	})

	t.Run("DataTypeString", func(t *testing.T) {
		// Valid string: max 20, actual 5, "Hello"
		data := []byte{20, 5, 'H', 'e', 'l', 'l', 'o', 0x00, 0x00}
		str, num := DecodeS7Value(data, DataTypeString, -1)
		if str != "Hello" || num != nil {
			t.Errorf("expected 'Hello', nil; got '%s', %v", str, num)
		}

		// String with trailing spaces
		dataSpaces := []byte{20, 7, 'H', 'e', 'l', 'l', 'o', ' ', ' '}
		str, num = DecodeS7Value(dataSpaces, DataTypeString, -1)
		if str != "Hello" || num != nil {
			t.Errorf("expected 'Hello', nil; got '%s', %v", str, num)
		}

		// String actual length exceeding data length
		dataShort := []byte{20, 10, 'A', 'B', 'C'}
		str, num = DecodeS7Value(dataShort, DataTypeString, -1)
		if str != "ABC" || num != nil {
			t.Errorf("expected 'ABC', nil; got '%s', %v", str, num)
		}

		// Truncated header
		str, num = DecodeS7Value([]byte{20}, DataTypeString, -1)
		if str != "" || num != nil {
			t.Errorf("expected '', nil; got '%s', %v", str, num)
		}
	})

	t.Run("Unknown data type fallback", func(t *testing.T) {
		str, num := DecodeS7Value([]byte{0x01, 0x02}, TagDataType("UnknownType"), -1)
		if str != "-" || num != nil {
			t.Errorf("expected '-', nil for unknown type; got '%s', %v", str, num)
		}
	})
}
