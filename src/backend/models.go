package backend

import (
	"github.com/google/uuid"
)

type PlcLinkSettings struct {
	Name        string `json:"name"`
	IpAddress   string `json:"ipAddress"`
	Rack        int    `json:"rack"`
	Slot        int    `json:"slot"`
	IsConnected bool   `json:"isConnected"`
}

func CreateDefaultPlcLinks() []PlcLinkSettings {
	return []PlcLinkSettings{
		{Name: "PLC1", IpAddress: "192.168.0.1", Rack: 0, Slot: 1},
	}
}

type TagDataType string

const (
	DataTypeBool  TagDataType = "Bool"
	DataTypeByte  TagDataType = "Byte"
	DataTypeWord  TagDataType = "Word"
	DataTypeInt   TagDataType = "Int"
	DataTypeDWord TagDataType = "DWord"
	DataTypeDInt  TagDataType = "DInt"
	DataTypeReal  TagDataType = "Real"
	DataTypeLReal TagDataType = "LReal"
)

const (
	MinPlcLinks       = 1
	MaxPlcLinks       = 8
	MaxTagsPerPlcLink = 16
)

type TagSettings struct {
	Id       uuid.UUID   `json:"id"`
	Name     string      `json:"name"`
	PlcLink  string      `json:"plcLink"`
	Address  string      `json:"address"`
	DataType TagDataType `json:"dataType"`
	YAxis    string      `json:"yAxis"`
	Color    string      `json:"color"`
	Enabled  bool        `json:"enabled"`
}

type YAxisSettings struct {
	Name      string  `json:"name"`
	Minimum   float64 `json:"minimum"`
	Maximum   float64 `json:"maximum"`
	AutoScale bool    `json:"autoScale"`
}

func CreateDefaultYAxes() []YAxisSettings {
	return []YAxisSettings{
		{Name: "Y-Axis 1", Minimum: 0, Maximum: 100, AutoScale: true},
		{Name: "Y-Axis 2", Minimum: 0, Maximum: 100, AutoScale: true},
		{Name: "Y-Axis 3", Minimum: 0, Maximum: 100, AutoScale: true},
		{Name: "Y-Axis 4", Minimum: 0, Maximum: 100, AutoScale: true},
	}
}

type InterpolationMode string

const (
	InterpolationLine         InterpolationMode = "Line"
	InterpolationDifferential InterpolationMode = "Differential"
)

type AppSettings struct {
	PollIntervalMs    int               `json:"pollIntervalMs"`
	TimeWindowSeconds int               `json:"timeWindowSeconds"`
	Interpolation     InterpolationMode `json:"interpolation"`
	PlcLinks          []PlcLinkSettings `json:"plcLinks"`
	Tags          []TagSettings `json:"tags"`
	YAxes             []YAxisSettings   `json:"yAxes"`
}

func CreateDefaultSettings() AppSettings {
	return AppSettings{
		PollIntervalMs:    100,
		TimeWindowSeconds: 60,
		Interpolation:     InterpolationLine,
		PlcLinks:          CreateDefaultPlcLinks(),
		Tags:          []TagSettings{},
		YAxes:             CreateDefaultYAxes(),
	}
}

type PollUpdate struct {
	TagId        uuid.UUID `json:"tagId"`
	Timestamp    string    `json:"timestamp"`
	Value        string    `json:"value"`
	NumericValue *float64  `json:"numericValue"`
	Quality      string    `json:"quality"`
	Error        string    `json:"error"`
}

// PollTiming reports the measured wall-clock period between poll cycles for a PLC link.
type PollTiming struct {
	PlcLink          string `json:"plcLink"`
	ActualIntervalMs int    `json:"actualIntervalMs"`
}

type SampleRecord struct {
	Timestamp string  `json:"timestamp"`
	TagName   string  `json:"tagName"`
	Address   string  `json:"address"`
	Value     float64 `json:"value"`
}
