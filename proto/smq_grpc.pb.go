// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// SuhaibMessageQueueClient is the client API for SuhaibMessageQueue service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type SuhaibMessageQueueClient interface {
	Connect(ctx context.Context, in *ConnectRequest, opts ...grpc.CallOption) (*ConnectResponse, error)
	GetLatestOffset(ctx context.Context, in *GetLatestOffsetRequest, opts ...grpc.CallOption) (*GetLatestOffsetResponse, error)
	GetEarliestOffset(ctx context.Context, in *GetLatestOffsetRequest, opts ...grpc.CallOption) (*GetLatestOffsetResponse, error)
	CreateTopic(ctx context.Context, in *CreateTopicRequest, opts ...grpc.CallOption) (*CreateTopicResponse, error)
	// Single message versions
	Produce(ctx context.Context, in *ProduceRequest, opts ...grpc.CallOption) (*ProduceResponse, error)
	Consume(ctx context.Context, in *ConsumeRequest, opts ...grpc.CallOption) (*ConsumeResponse, error)
	// Stream versions
	StreamProduce(ctx context.Context, opts ...grpc.CallOption) (SuhaibMessageQueue_StreamProduceClient, error)
	StreamConsume(ctx context.Context, in *ConsumeRequest, opts ...grpc.CallOption) (SuhaibMessageQueue_StreamConsumeClient, error)
	DeleteUntilOffset(ctx context.Context, in *DeleteUntilOffsetRequest, opts ...grpc.CallOption) (*DeleteUntilOffsetResponse, error)
}

type suhaibMessageQueueClient struct {
	cc grpc.ClientConnInterface
}

func NewSuhaibMessageQueueClient(cc grpc.ClientConnInterface) SuhaibMessageQueueClient {
	return &suhaibMessageQueueClient{cc}
}

func (c *suhaibMessageQueueClient) Connect(ctx context.Context, in *ConnectRequest, opts ...grpc.CallOption) (*ConnectResponse, error) {
	out := new(ConnectResponse)
	err := c.cc.Invoke(ctx, "/smq.SuhaibMessageQueue/Connect", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *suhaibMessageQueueClient) GetLatestOffset(ctx context.Context, in *GetLatestOffsetRequest, opts ...grpc.CallOption) (*GetLatestOffsetResponse, error) {
	out := new(GetLatestOffsetResponse)
	err := c.cc.Invoke(ctx, "/smq.SuhaibMessageQueue/GetLatestOffset", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *suhaibMessageQueueClient) GetEarliestOffset(ctx context.Context, in *GetLatestOffsetRequest, opts ...grpc.CallOption) (*GetLatestOffsetResponse, error) {
	out := new(GetLatestOffsetResponse)
	err := c.cc.Invoke(ctx, "/smq.SuhaibMessageQueue/GetEarliestOffset", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *suhaibMessageQueueClient) CreateTopic(ctx context.Context, in *CreateTopicRequest, opts ...grpc.CallOption) (*CreateTopicResponse, error) {
	out := new(CreateTopicResponse)
	err := c.cc.Invoke(ctx, "/smq.SuhaibMessageQueue/CreateTopic", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *suhaibMessageQueueClient) Produce(ctx context.Context, in *ProduceRequest, opts ...grpc.CallOption) (*ProduceResponse, error) {
	out := new(ProduceResponse)
	err := c.cc.Invoke(ctx, "/smq.SuhaibMessageQueue/Produce", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *suhaibMessageQueueClient) Consume(ctx context.Context, in *ConsumeRequest, opts ...grpc.CallOption) (*ConsumeResponse, error) {
	out := new(ConsumeResponse)
	err := c.cc.Invoke(ctx, "/smq.SuhaibMessageQueue/Consume", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *suhaibMessageQueueClient) StreamProduce(ctx context.Context, opts ...grpc.CallOption) (SuhaibMessageQueue_StreamProduceClient, error) {
	stream, err := c.cc.NewStream(ctx, &SuhaibMessageQueue_ServiceDesc.Streams[0], "/smq.SuhaibMessageQueue/StreamProduce", opts...)
	if err != nil {
		return nil, err
	}
	x := &suhaibMessageQueueStreamProduceClient{stream}
	return x, nil
}

type SuhaibMessageQueue_StreamProduceClient interface {
	Send(*ProduceRequest) error
	CloseAndRecv() (*ProduceResponse, error)
	grpc.ClientStream
}

type suhaibMessageQueueStreamProduceClient struct {
	grpc.ClientStream
}

func (x *suhaibMessageQueueStreamProduceClient) Send(m *ProduceRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *suhaibMessageQueueStreamProduceClient) CloseAndRecv() (*ProduceResponse, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(ProduceResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *suhaibMessageQueueClient) StreamConsume(ctx context.Context, in *ConsumeRequest, opts ...grpc.CallOption) (SuhaibMessageQueue_StreamConsumeClient, error) {
	stream, err := c.cc.NewStream(ctx, &SuhaibMessageQueue_ServiceDesc.Streams[1], "/smq.SuhaibMessageQueue/StreamConsume", opts...)
	if err != nil {
		return nil, err
	}
	x := &suhaibMessageQueueStreamConsumeClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type SuhaibMessageQueue_StreamConsumeClient interface {
	Recv() (*ConsumeResponse, error)
	grpc.ClientStream
}

type suhaibMessageQueueStreamConsumeClient struct {
	grpc.ClientStream
}

func (x *suhaibMessageQueueStreamConsumeClient) Recv() (*ConsumeResponse, error) {
	m := new(ConsumeResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *suhaibMessageQueueClient) DeleteUntilOffset(ctx context.Context, in *DeleteUntilOffsetRequest, opts ...grpc.CallOption) (*DeleteUntilOffsetResponse, error) {
	out := new(DeleteUntilOffsetResponse)
	err := c.cc.Invoke(ctx, "/smq.SuhaibMessageQueue/DeleteUntilOffset", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SuhaibMessageQueueServer is the server API for SuhaibMessageQueue service.
// All implementations must embed UnimplementedSuhaibMessageQueueServer
// for forward compatibility
type SuhaibMessageQueueServer interface {
	Connect(context.Context, *ConnectRequest) (*ConnectResponse, error)
	GetLatestOffset(context.Context, *GetLatestOffsetRequest) (*GetLatestOffsetResponse, error)
	GetEarliestOffset(context.Context, *GetLatestOffsetRequest) (*GetLatestOffsetResponse, error)
	CreateTopic(context.Context, *CreateTopicRequest) (*CreateTopicResponse, error)
	// Single message versions
	Produce(context.Context, *ProduceRequest) (*ProduceResponse, error)
	Consume(context.Context, *ConsumeRequest) (*ConsumeResponse, error)
	// Stream versions
	StreamProduce(SuhaibMessageQueue_StreamProduceServer) error
	StreamConsume(*ConsumeRequest, SuhaibMessageQueue_StreamConsumeServer) error
	DeleteUntilOffset(context.Context, *DeleteUntilOffsetRequest) (*DeleteUntilOffsetResponse, error)
	mustEmbedUnimplementedSuhaibMessageQueueServer()
}

// UnimplementedSuhaibMessageQueueServer must be embedded to have forward compatible implementations.
type UnimplementedSuhaibMessageQueueServer struct {
}

func (UnimplementedSuhaibMessageQueueServer) Connect(context.Context, *ConnectRequest) (*ConnectResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Connect not implemented")
}
func (UnimplementedSuhaibMessageQueueServer) GetLatestOffset(context.Context, *GetLatestOffsetRequest) (*GetLatestOffsetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetLatestOffset not implemented")
}
func (UnimplementedSuhaibMessageQueueServer) GetEarliestOffset(context.Context, *GetLatestOffsetRequest) (*GetLatestOffsetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetEarliestOffset not implemented")
}
func (UnimplementedSuhaibMessageQueueServer) CreateTopic(context.Context, *CreateTopicRequest) (*CreateTopicResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateTopic not implemented")
}
func (UnimplementedSuhaibMessageQueueServer) Produce(context.Context, *ProduceRequest) (*ProduceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Produce not implemented")
}
func (UnimplementedSuhaibMessageQueueServer) Consume(context.Context, *ConsumeRequest) (*ConsumeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Consume not implemented")
}
func (UnimplementedSuhaibMessageQueueServer) StreamProduce(SuhaibMessageQueue_StreamProduceServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamProduce not implemented")
}
func (UnimplementedSuhaibMessageQueueServer) StreamConsume(*ConsumeRequest, SuhaibMessageQueue_StreamConsumeServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamConsume not implemented")
}
func (UnimplementedSuhaibMessageQueueServer) DeleteUntilOffset(context.Context, *DeleteUntilOffsetRequest) (*DeleteUntilOffsetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteUntilOffset not implemented")
}
func (UnimplementedSuhaibMessageQueueServer) mustEmbedUnimplementedSuhaibMessageQueueServer() {}

// UnsafeSuhaibMessageQueueServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to SuhaibMessageQueueServer will
// result in compilation errors.
type UnsafeSuhaibMessageQueueServer interface {
	mustEmbedUnimplementedSuhaibMessageQueueServer()
}

func RegisterSuhaibMessageQueueServer(s grpc.ServiceRegistrar, srv SuhaibMessageQueueServer) {
	s.RegisterService(&SuhaibMessageQueue_ServiceDesc, srv)
}

func _SuhaibMessageQueue_Connect_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ConnectRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SuhaibMessageQueueServer).Connect(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/smq.SuhaibMessageQueue/Connect",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SuhaibMessageQueueServer).Connect(ctx, req.(*ConnectRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _SuhaibMessageQueue_GetLatestOffset_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetLatestOffsetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SuhaibMessageQueueServer).GetLatestOffset(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/smq.SuhaibMessageQueue/GetLatestOffset",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SuhaibMessageQueueServer).GetLatestOffset(ctx, req.(*GetLatestOffsetRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _SuhaibMessageQueue_GetEarliestOffset_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetLatestOffsetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SuhaibMessageQueueServer).GetEarliestOffset(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/smq.SuhaibMessageQueue/GetEarliestOffset",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SuhaibMessageQueueServer).GetEarliestOffset(ctx, req.(*GetLatestOffsetRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _SuhaibMessageQueue_CreateTopic_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateTopicRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SuhaibMessageQueueServer).CreateTopic(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/smq.SuhaibMessageQueue/CreateTopic",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SuhaibMessageQueueServer).CreateTopic(ctx, req.(*CreateTopicRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _SuhaibMessageQueue_Produce_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ProduceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SuhaibMessageQueueServer).Produce(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/smq.SuhaibMessageQueue/Produce",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SuhaibMessageQueueServer).Produce(ctx, req.(*ProduceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _SuhaibMessageQueue_Consume_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ConsumeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SuhaibMessageQueueServer).Consume(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/smq.SuhaibMessageQueue/Consume",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SuhaibMessageQueueServer).Consume(ctx, req.(*ConsumeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _SuhaibMessageQueue_StreamProduce_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(SuhaibMessageQueueServer).StreamProduce(&suhaibMessageQueueStreamProduceServer{stream})
}

type SuhaibMessageQueue_StreamProduceServer interface {
	SendAndClose(*ProduceResponse) error
	Recv() (*ProduceRequest, error)
	grpc.ServerStream
}

type suhaibMessageQueueStreamProduceServer struct {
	grpc.ServerStream
}

func (x *suhaibMessageQueueStreamProduceServer) SendAndClose(m *ProduceResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *suhaibMessageQueueStreamProduceServer) Recv() (*ProduceRequest, error) {
	m := new(ProduceRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _SuhaibMessageQueue_StreamConsume_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(ConsumeRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(SuhaibMessageQueueServer).StreamConsume(m, &suhaibMessageQueueStreamConsumeServer{stream})
}

type SuhaibMessageQueue_StreamConsumeServer interface {
	Send(*ConsumeResponse) error
	grpc.ServerStream
}

type suhaibMessageQueueStreamConsumeServer struct {
	grpc.ServerStream
}

func (x *suhaibMessageQueueStreamConsumeServer) Send(m *ConsumeResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _SuhaibMessageQueue_DeleteUntilOffset_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteUntilOffsetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SuhaibMessageQueueServer).DeleteUntilOffset(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/smq.SuhaibMessageQueue/DeleteUntilOffset",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SuhaibMessageQueueServer).DeleteUntilOffset(ctx, req.(*DeleteUntilOffsetRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// SuhaibMessageQueue_ServiceDesc is the grpc.ServiceDesc for SuhaibMessageQueue service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var SuhaibMessageQueue_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "smq.SuhaibMessageQueue",
	HandlerType: (*SuhaibMessageQueueServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Connect",
			Handler:    _SuhaibMessageQueue_Connect_Handler,
		},
		{
			MethodName: "GetLatestOffset",
			Handler:    _SuhaibMessageQueue_GetLatestOffset_Handler,
		},
		{
			MethodName: "GetEarliestOffset",
			Handler:    _SuhaibMessageQueue_GetEarliestOffset_Handler,
		},
		{
			MethodName: "CreateTopic",
			Handler:    _SuhaibMessageQueue_CreateTopic_Handler,
		},
		{
			MethodName: "Produce",
			Handler:    _SuhaibMessageQueue_Produce_Handler,
		},
		{
			MethodName: "Consume",
			Handler:    _SuhaibMessageQueue_Consume_Handler,
		},
		{
			MethodName: "DeleteUntilOffset",
			Handler:    _SuhaibMessageQueue_DeleteUntilOffset_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamProduce",
			Handler:       _SuhaibMessageQueue_StreamProduce_Handler,
			ClientStreams: true,
		},
		{
			StreamName:    "StreamConsume",
			Handler:       _SuhaibMessageQueue_StreamConsume_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "smq.proto",
}
