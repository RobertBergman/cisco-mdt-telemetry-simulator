// Package mdt_dialout implements the gRPC MDT dial-out streaming service
package mdt_dialout

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protowire"
)

// MdtDialoutArgs represents the dial-out streaming message
type MdtDialoutArgs struct {
	ReqId  int64
	Data   []byte
	Errors string
}

// Marshal encodes MdtDialoutArgs to protobuf wire format
func (m *MdtDialoutArgs) Marshal() ([]byte, error) {
	var buf []byte

	// Field 1: ReqId (int64)
	if m.ReqId != 0 {
		buf = protowire.AppendTag(buf, 1, protowire.VarintType)
		buf = protowire.AppendVarint(buf, uint64(m.ReqId))
	}

	// Field 2: data (bytes)
	if len(m.Data) > 0 {
		buf = protowire.AppendTag(buf, 2, protowire.BytesType)
		buf = protowire.AppendBytes(buf, m.Data)
	}

	// Field 3: errors (string)
	if m.Errors != "" {
		buf = protowire.AppendTag(buf, 3, protowire.BytesType)
		buf = protowire.AppendString(buf, m.Errors)
	}

	return buf, nil
}

// GRPCMdtDialoutClient is the client interface for MDT dial-out
type GRPCMdtDialoutClient interface {
	MdtDialout(ctx context.Context, opts ...grpc.CallOption) (MdtDialout_MdtDialoutClient, error)
}

// MdtDialout_MdtDialoutClient is the streaming interface
type MdtDialout_MdtDialoutClient interface {
	Send(*MdtDialoutArgs) error
	CloseAndRecv() (*MdtDialoutArgs, error)
	grpc.ClientStream
}

// grpcMdtDialoutClient implements GRPCMdtDialoutClient
type grpcMdtDialoutClient struct {
	cc grpc.ClientConnInterface
}

// NewGRPCMdtDialoutClient creates a new dial-out client
func NewGRPCMdtDialoutClient(cc grpc.ClientConnInterface) GRPCMdtDialoutClient {
	return &grpcMdtDialoutClient{cc}
}

func (c *grpcMdtDialoutClient) MdtDialout(ctx context.Context, opts ...grpc.CallOption) (MdtDialout_MdtDialoutClient, error) {
	stream, err := c.cc.NewStream(ctx, &mdtDialoutServiceDesc.Streams[0], "/mdt_dialout.gRPCMdtDialout/MdtDialout", opts...)
	if err != nil {
		return nil, err
	}
	return &mdtDialoutMdtDialoutClient{stream}, nil
}

type mdtDialoutMdtDialoutClient struct {
	grpc.ClientStream
}

func (x *mdtDialoutMdtDialoutClient) Send(m *MdtDialoutArgs) error {
	data, err := m.Marshal()
	if err != nil {
		return err
	}
	return x.ClientStream.SendMsg(&rawMessage{data: data})
}

func (x *mdtDialoutMdtDialoutClient) CloseAndRecv() (*MdtDialoutArgs, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := &rawMessage{}
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	// Parse response - for now just return empty
	return &MdtDialoutArgs{}, nil
}

// rawMessage is a helper for sending pre-encoded protobuf data
type rawMessage struct {
	data []byte
}

func (m *rawMessage) Reset()        {}
func (m *rawMessage) String() string { return string(m.data) }
func (m *rawMessage) ProtoMessage() {}

func (m *rawMessage) Marshal() ([]byte, error) {
	return m.data, nil
}

func (m *rawMessage) Unmarshal(b []byte) error {
	m.data = b
	return nil
}

// Service descriptor for gRPC
var mdtDialoutServiceDesc = grpc.ServiceDesc{
	ServiceName: "mdt_dialout.gRPCMdtDialout",
	HandlerType: (*GRPCMdtDialoutClient)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "MdtDialout",
			ClientStreams: true,
		},
	},
	Metadata: "mdt_dialout.proto",
}
