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
		// Memory (Merker) addresses
		{
			name:         "Memory bit M0.0",
			address:      "M0.0",
			valueType:    DataTypeBool,
			expectedArea: s7AreaMK,
			expectedDb:   0,
			expectedByte: 0,
			expectedBit:  0,
			expectedLen:  1,
		},
		{
			name:         "Memory bit MX10.7",
			address:      "MX10.7",
			valueType:    DataTypeBool,
			expectedArea: s7AreaMK,
			expectedDb:   0,
			expectedByte: 10,
			expectedBit:  7,
			expectedLen:  1,
		},
		{
			name:         "Memory Byte MB5",
			address:      "MB5",
			valueType:    DataTypeByte,
			expectedArea: s7AreaMK,
			expectedDb:   0,
			expectedByte: 5,
			expectedBit:  -1,
			expectedLen:  1,
		},
		{
			name:         "Memory Word MW100",
			address:      "MW100",
			valueType:    DataTypeWord,
			expectedArea: s7AreaMK,
			expectedDb:   0,
			expectedByte: 100,
			expectedBit:  -1,
			expectedLen:  2,
		},
		{
			name:         "Memory Int MW102",
			address:      "MW102",
			valueType:    DataTypeInt,
			expectedArea: s7AreaMK,
			expectedDb:   0,
			expectedByte: 102,
			expectedBit:  -1,
			expectedLen:  2,
		},
		{
			name:         "Memory DWord MD200",
			address:      "MD200",
			valueType:    DataTypeDWord,
			expectedArea: s7AreaMK,
			expectedDb:   0,
			expectedByte: 200,
			expectedBit:  -1,
			expectedLen:  4,
		},
		{
			name:         "Memory Real MD204",
			address:      "MD204",
			valueType:    DataTypeReal,
			expectedArea: s7AreaMK,
			expectedDb:   0,
			expectedByte: 204,
			expectedBit:  -1,
			expectedLen:  4,
		},
		// Input (PE) addresses
		{
			name:         "Input bit I0.5",
			address:      "I0.5",
			valueType:    DataTypeBool,
			expectedArea: s7AreaPE,
			expectedDb:   0,
			expectedByte: 0,
			expectedBit:  5,
			expectedLen:  1,
		},
		{
			name:         "Input bit IX1.0",
			address:      "IX1.0",
			valueType:    DataTypeBool,
			expectedArea: s7AreaPE,
			expectedDb:   0,
			expectedByte: 1,
			expectedBit:  0,
			expectedLen:  1,
		},
		{
			name:         "Input Byte IB0",
			address:      "IB0",
			valueType:    DataTypeByte,
			expectedArea: s7AreaPE,
			expectedDb:   0,
			expectedByte: 0,
			expectedBit:  -1,
			expectedLen:  1,
		},
		{
			name:         "Input Word IW4",
			address:      "IW4",
			valueType:    DataTypeInt,
			expectedArea: s7AreaPE,
			expectedDb:   0,
			expectedByte: 4,
			expectedBit:  -1,
			expectedLen:  2,
		},
		{
			name:         "Input DWord ID8",
			address:      "ID8",
			valueType:    DataTypeDWord,
			expectedArea: s7AreaPE,
			expectedDb:   0,
			expectedByte: 8,
			expectedBit:  -1,
			expectedLen:  4,
		},
		// Output (PA) addresses
		{
			name:         "Output bit Q0.0",
			address:      "Q0.0",
			valueType:    DataTypeBool,
			expectedArea: s7AreaPA,
			expectedDb:   0,
			expectedByte: 0,
			expectedBit:  0,
			expectedLen:  1,
		},
		{
			name:         "Output bit QX2.3",
			address:      "QX2.3",
			valueType:    DataTypeBool,
			expectedArea: s7AreaPA,
			expectedDb:   0,
			expectedByte: 2,
			expectedBit:  3,
			expectedLen:  1,
		},
		{
			name:         "Output Byte QB1",
			address:      "QB1",
			valueType:    DataTypeByte,
			expectedArea: s7AreaPA,
			expectedDb:   0,
			expectedByte: 1,
			expectedBit:  -1,
			expectedLen:  1,
		},
		{
			name:         "Output Word QW10",
			address:      "QW10",
			valueType:    DataTypeWord,
			expectedArea: s7AreaPA,
			expectedDb:   0,
			expectedByte: 10,
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
		// Timer (TM) addresses
		{
			name:         "Timer T0",
			address:      "T0",
			valueType:    DataTypeWord,
			expectedArea: s7AreaTM,
			expectedDb:   0,
			expectedByte: 0,
			expectedBit:  -1,
			expectedLen:  2,
		},
		{
			name:         "Timer TM12",
			address:      "TM12",
			valueType:    DataTypeWord,
			expectedArea: s7AreaTM,
			expectedDb:   0,
			expectedByte: 12,
			expectedBit:  -1,
			expectedLen:  2,
		},
		// Counter (CT) addresses
		{
			name:         "Counter C0",
			address:      "C0",
			valueType:    DataTypeInt,
			expectedArea: s7AreaCT,
			expectedDb:   0,
			expectedByte: 0,
			expectedBit:  -1,
			expectedLen:  2,
		},
		{
			name:         "Counter Z5 (German Zähler)",
			address:      "Z5",
			valueType:    DataTypeInt,
			expectedArea: s7AreaCT,
			expectedDb:   0,
			expectedByte: 5,
			expectedBit:  -1,
			expectedLen:  2,
		},
		{
			name:         "Counter CT20",
			address:      "CT20",
			valueType:    DataTypeInt,
			expectedArea: s7AreaCT,
			expectedDb:   0,
			expectedByte: 20,
			expectedBit:  -1,
			expectedLen:  2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := ParseS7Address(tc.address, tc.valueType)
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
		name        string
		address     string
		valueType   TagDataType
		errContains string
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
			spec, err := ParseS7Address(tc.address, tc.valueType)
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
		expected int
	}{
		{DataTypeBool, 1},
		{DataTypeByte, 1},
		{DataTypeWord, 2},
		{DataTypeInt, 2},
		{DataTypeDWord, 4},
		{DataTypeDInt, 4},
		{DataTypeReal, 4},
		{DataTypeLReal, 8},
		{TagDataType("Unknown"), 4}, // default fallback
	}

	for _, tc := range tests {
		t.Run(string(tc.valType), func(t *testing.T) {
			got := getByteLength(tc.valType)
			if got != tc.expected {
				t.Errorf("getByteLength(%s) = %d; expected %d", tc.valType, got, tc.expected)
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

	t.Run("Unknown data type fallback", func(t *testing.T) {
		str, num := DecodeS7Value([]byte{0x01, 0x02}, TagDataType("UnknownType"), -1)
		if str != "-" || num != nil {
			t.Errorf("expected '-', nil for unknown type; got '%s', %v", str, num)
		}
	})
}

func TestDecodeS7Timer(t *testing.T) {
	t.Run("Empty or truncated data", func(t *testing.T) {
		str, num := DecodeS7Timer([]byte{})
		if str != "-" || num != nil {
			t.Errorf("expected '-', nil; got '%s', %v", str, num)
		}
		str, num = DecodeS7Timer([]byte{0x20})
		if str != "-" || num != nil {
			t.Errorf("expected '-', nil; got '%s', %v", str, num)
		}
	})

	t.Run("10ms base (multiplier 0.01)", func(t *testing.T) {
		// BCD 123 with base 0 (10ms) -> 0x01, 0x23 -> 1.230s
		str, num := DecodeS7Timer([]byte{0x01, 0x23})
		if str != "1.230" || num == nil || *num != 1.23 {
			t.Errorf("expected '1.230', 1.23; got '%s', %v", str, num)
		}
	})

	t.Run("100ms base (multiplier 0.1)", func(t *testing.T) {
		// BCD 456 with base 1 (100ms) -> 0x14, 0x56 -> 45.600s
		str, num := DecodeS7Timer([]byte{0x14, 0x56})
		if str != "45.600" || num == nil || *num != 45.6 {
			t.Errorf("expected '45.600', 45.6; got '%s', %v", str, num)
		}
	})

	t.Run("1s base (multiplier 1.0)", func(t *testing.T) {
		// BCD 120 with base 2 (1s) -> 0x21, 0x20 -> 120.000s
		str, num := DecodeS7Timer([]byte{0x21, 0x20})
		if str != "120.000" || num == nil || *num != 120.0 {
			t.Errorf("expected '120.000', 120.0; got '%s', %v", str, num)
		}
	})

	t.Run("10s base (multiplier 10.0)", func(t *testing.T) {
		// BCD 050 with base 3 (10s) -> 0x30, 0x50 -> 500.000s
		str, num := DecodeS7Timer([]byte{0x30, 0x50})
		if str != "500.000" || num == nil || *num != 500.0 {
			t.Errorf("expected '500.000', 500.0; got '%s', %v", str, num)
		}
	})
}

func TestDecodeS7Counter(t *testing.T) {
	t.Run("Empty or truncated data", func(t *testing.T) {
		str, num := DecodeS7Counter([]byte{})
		if str != "-" || num != nil {
			t.Errorf("expected '-', nil; got '%s', %v", str, num)
		}
		str, num = DecodeS7Counter([]byte{0x05})
		if str != "-" || num != nil {
			t.Errorf("expected '-', nil; got '%s', %v", str, num)
		}
	})

	t.Run("BCD Counter decoding", func(t *testing.T) {
		// BCD 123 -> 0x01, 0x23
		str, num := DecodeS7Counter([]byte{0x01, 0x23})
		if str != "123" || num == nil || *num != 123.0 {
			t.Errorf("expected '123', 123.0; got '%s', %v", str, num)
		}

		// BCD 999 -> 0x09, 0x99
		str, num = DecodeS7Counter([]byte{0x09, 0x99})
		if str != "999" || num == nil || *num != 999.0 {
			t.Errorf("expected '999', 999.0; got '%s', %v", str, num)
		}
	})
}
