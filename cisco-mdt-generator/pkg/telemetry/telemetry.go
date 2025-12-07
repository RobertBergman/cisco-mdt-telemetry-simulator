// Package telemetry contains the Cisco MDT telemetry message types
// This is a manual implementation matching the Cisco telemetry.proto schema
package telemetry

import (
	"math"

	"google.golang.org/protobuf/encoding/protowire"
)

// Telemetry represents a GPB-KV telemetry message
type Telemetry struct {
	NodeIDStr           string
	SubscriptionIDStr   string
	EncodingPath        string
	CollectionID        uint64
	CollectionStartTime uint64
	MsgTimestamp        uint64
	CollectionEndTime   uint64
	DataGpbkv           []*TelemetryField
}

// TelemetryField represents a field in the telemetry tree
type TelemetryField struct {
	Timestamp   uint64
	Name        string
	Fields      []*TelemetryField
	StringValue *string
	Uint32Value *uint32
	Uint64Value *uint64
	BoolValue   *bool
	DoubleValue *float64
	FloatValue  *float32
	BytesValue  []byte
	Sint32Value *int32
	Sint64Value *int64
}

// Marshal encodes the Telemetry message to protobuf wire format
func (t *Telemetry) Marshal() ([]byte, error) {
	var buf []byte

	// Field 1: node_id_str (string)
	if t.NodeIDStr != "" {
		buf = protowire.AppendTag(buf, 1, protowire.BytesType)
		buf = protowire.AppendString(buf, t.NodeIDStr)
	}

	// Field 3: subscription_id_str (string)
	if t.SubscriptionIDStr != "" {
		buf = protowire.AppendTag(buf, 3, protowire.BytesType)
		buf = protowire.AppendString(buf, t.SubscriptionIDStr)
	}

	// Field 6: encoding_path (string)
	if t.EncodingPath != "" {
		buf = protowire.AppendTag(buf, 6, protowire.BytesType)
		buf = protowire.AppendString(buf, t.EncodingPath)
	}

	// Field 8: collection_id (uint64)
	if t.CollectionID != 0 {
		buf = protowire.AppendTag(buf, 8, protowire.VarintType)
		buf = protowire.AppendVarint(buf, t.CollectionID)
	}

	// Field 9: collection_start_time (uint64)
	if t.CollectionStartTime != 0 {
		buf = protowire.AppendTag(buf, 9, protowire.VarintType)
		buf = protowire.AppendVarint(buf, t.CollectionStartTime)
	}

	// Field 10: msg_timestamp (uint64)
	if t.MsgTimestamp != 0 {
		buf = protowire.AppendTag(buf, 10, protowire.VarintType)
		buf = protowire.AppendVarint(buf, t.MsgTimestamp)
	}

	// Field 11: data_gpbkv (repeated TelemetryField)
	for _, field := range t.DataGpbkv {
		fieldBytes, err := field.Marshal()
		if err != nil {
			return nil, err
		}
		buf = protowire.AppendTag(buf, 11, protowire.BytesType)
		buf = protowire.AppendBytes(buf, fieldBytes)
	}

	// Field 13: collection_end_time (uint64)
	if t.CollectionEndTime != 0 {
		buf = protowire.AppendTag(buf, 13, protowire.VarintType)
		buf = protowire.AppendVarint(buf, t.CollectionEndTime)
	}

	return buf, nil
}

// Marshal encodes a TelemetryField to protobuf wire format
func (f *TelemetryField) Marshal() ([]byte, error) {
	var buf []byte

	// Field 1: timestamp (uint64)
	if f.Timestamp != 0 {
		buf = protowire.AppendTag(buf, 1, protowire.VarintType)
		buf = protowire.AppendVarint(buf, f.Timestamp)
	}

	// Field 2: name (string)
	if f.Name != "" {
		buf = protowire.AppendTag(buf, 2, protowire.BytesType)
		buf = protowire.AppendString(buf, f.Name)
	}

	// Value fields (oneof - only one should be set)
	// Field 4: bytes_value
	if f.BytesValue != nil {
		buf = protowire.AppendTag(buf, 4, protowire.BytesType)
		buf = protowire.AppendBytes(buf, f.BytesValue)
	}

	// Field 5: string_value
	if f.StringValue != nil {
		buf = protowire.AppendTag(buf, 5, protowire.BytesType)
		buf = protowire.AppendString(buf, *f.StringValue)
	}

	// Field 6: bool_value
	if f.BoolValue != nil {
		buf = protowire.AppendTag(buf, 6, protowire.VarintType)
		val := uint64(0)
		if *f.BoolValue {
			val = 1
		}
		buf = protowire.AppendVarint(buf, val)
	}

	// Field 7: uint32_value
	if f.Uint32Value != nil {
		buf = protowire.AppendTag(buf, 7, protowire.VarintType)
		buf = protowire.AppendVarint(buf, uint64(*f.Uint32Value))
	}

	// Field 8: uint64_value
	if f.Uint64Value != nil {
		buf = protowire.AppendTag(buf, 8, protowire.VarintType)
		buf = protowire.AppendVarint(buf, *f.Uint64Value)
	}

	// Field 9: sint32_value
	if f.Sint32Value != nil {
		buf = protowire.AppendTag(buf, 9, protowire.VarintType)
		buf = protowire.AppendVarint(buf, protowire.EncodeZigZag(int64(*f.Sint32Value)))
	}

	// Field 10: sint64_value
	if f.Sint64Value != nil {
		buf = protowire.AppendTag(buf, 10, protowire.VarintType)
		buf = protowire.AppendVarint(buf, protowire.EncodeZigZag(*f.Sint64Value))
	}

	// Field 11: double_value (fixed64)
	if f.DoubleValue != nil {
		buf = protowire.AppendTag(buf, 11, protowire.Fixed64Type)
		buf = protowire.AppendFixed64(buf, math.Float64bits(*f.DoubleValue))
	}

	// Field 12: float_value (fixed32)
	if f.FloatValue != nil {
		buf = protowire.AppendTag(buf, 12, protowire.Fixed32Type)
		buf = protowire.AppendFixed32(buf, math.Float32bits(*f.FloatValue))
	}

	// Field 15: fields (repeated TelemetryField)
	for _, child := range f.Fields {
		childBytes, err := child.Marshal()
		if err != nil {
			return nil, err
		}
		buf = protowire.AppendTag(buf, 15, protowire.BytesType)
		buf = protowire.AppendBytes(buf, childBytes)
	}

	return buf, nil
}

// Helper functions for creating TelemetryField values

func StringField(name string, value string, ts uint64) *TelemetryField {
	return &TelemetryField{
		Name:        name,
		Timestamp:   ts,
		StringValue: &value,
	}
}

func Uint32Field(name string, value uint32, ts uint64) *TelemetryField {
	return &TelemetryField{
		Name:        name,
		Timestamp:   ts,
		Uint32Value: &value,
	}
}

func Uint64Field(name string, value uint64, ts uint64) *TelemetryField {
	return &TelemetryField{
		Name:        name,
		Timestamp:   ts,
		Uint64Value: &value,
	}
}

func ContainerField(name string, children []*TelemetryField, ts uint64) *TelemetryField {
	return &TelemetryField{
		Name:      name,
		Timestamp: ts,
		Fields:    children,
	}
}

// RowField creates a "row" container that matches NX-OS telemetry structure
// with "keys" and "content" sub-fields that Telegraf expects
func RowField(keys []*TelemetryField, content []*TelemetryField, ts uint64) *TelemetryField {
	return &TelemetryField{
		Timestamp: ts,
		Fields: []*TelemetryField{
			{
				Name:      "keys",
				Timestamp: ts,
				Fields:    keys,
			},
			{
				Name:      "content",
				Timestamp: ts,
				Fields:    content,
			},
		},
	}
}
