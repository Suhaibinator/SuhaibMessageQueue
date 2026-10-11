// Package proto preserves the original v1 generated-code import path.
//
// Deprecated: import github.com/Suhaibinator/SuhaibMessageQueue/api/smq/v1
// and alias it as smqv1 in new code.
package proto

import (
	smqv1 "github.com/Suhaibinator/SuhaibMessageQueue/api/smq/v1"
	"google.golang.org/grpc"
)

const (
	SuhaibMessageQueue_Connect_FullMethodName           = smqv1.SuhaibMessageQueue_Connect_FullMethodName
	SuhaibMessageQueue_GetLatestOffset_FullMethodName   = smqv1.SuhaibMessageQueue_GetLatestOffset_FullMethodName
	SuhaibMessageQueue_GetEarliestOffset_FullMethodName = smqv1.SuhaibMessageQueue_GetEarliestOffset_FullMethodName
	SuhaibMessageQueue_CreateTopic_FullMethodName       = smqv1.SuhaibMessageQueue_CreateTopic_FullMethodName
	SuhaibMessageQueue_Produce_FullMethodName           = smqv1.SuhaibMessageQueue_Produce_FullMethodName
	SuhaibMessageQueue_Consume_FullMethodName           = smqv1.SuhaibMessageQueue_Consume_FullMethodName
	SuhaibMessageQueue_StreamProduce_FullMethodName     = smqv1.SuhaibMessageQueue_StreamProduce_FullMethodName
	SuhaibMessageQueue_StreamConsume_FullMethodName     = smqv1.SuhaibMessageQueue_StreamConsume_FullMethodName
	SuhaibMessageQueue_DeleteUntilOffset_FullMethodName = smqv1.SuhaibMessageQueue_DeleteUntilOffset_FullMethodName
	SuhaibMessageQueue_BulkRetrieve_FullMethodName      = smqv1.SuhaibMessageQueue_BulkRetrieve_FullMethodName
)

type (
	RetrievedMessage          = smqv1.RetrievedMessage
	BulkRetrieveRequest       = smqv1.BulkRetrieveRequest
	BulkRetrieveResponse      = smqv1.BulkRetrieveResponse
	ConnectRequest            = smqv1.ConnectRequest
	ConnectResponse           = smqv1.ConnectResponse
	GetLatestOffsetRequest    = smqv1.GetLatestOffsetRequest
	GetLatestOffsetResponse   = smqv1.GetLatestOffsetResponse
	GetEarliestOffsetRequest  = smqv1.GetEarliestOffsetRequest
	GetEarliestOffsetResponse = smqv1.GetEarliestOffsetResponse
	CreateTopicRequest        = smqv1.CreateTopicRequest
	CreateTopicResponse       = smqv1.CreateTopicResponse
	ProduceRequest            = smqv1.ProduceRequest
	ProduceResponse           = smqv1.ProduceResponse
	ConsumeRequest            = smqv1.ConsumeRequest
	ConsumeResponse           = smqv1.ConsumeResponse
	DeleteUntilOffsetRequest  = smqv1.DeleteUntilOffsetRequest
	DeleteUntilOffsetResponse = smqv1.DeleteUntilOffsetResponse

	SuhaibMessageQueueClient               = smqv1.SuhaibMessageQueueClient
	SuhaibMessageQueueServer               = smqv1.SuhaibMessageQueueServer
	UnimplementedSuhaibMessageQueueServer  = smqv1.UnimplementedSuhaibMessageQueueServer
	UnsafeSuhaibMessageQueueServer         = smqv1.UnsafeSuhaibMessageQueueServer
	SuhaibMessageQueue_StreamProduceClient = smqv1.SuhaibMessageQueue_StreamProduceClient
	SuhaibMessageQueue_StreamConsumeClient = smqv1.SuhaibMessageQueue_StreamConsumeClient
	SuhaibMessageQueue_StreamProduceServer = smqv1.SuhaibMessageQueue_StreamProduceServer
	SuhaibMessageQueue_StreamConsumeServer = smqv1.SuhaibMessageQueue_StreamConsumeServer
)

var (
	File_smq_proto                 = smqv1.File_smq_proto
	SuhaibMessageQueue_ServiceDesc = smqv1.SuhaibMessageQueue_ServiceDesc
)

func NewSuhaibMessageQueueClient(cc grpc.ClientConnInterface) SuhaibMessageQueueClient {
	return smqv1.NewSuhaibMessageQueueClient(cc)
}

func RegisterSuhaibMessageQueueServer(registrar grpc.ServiceRegistrar, server SuhaibMessageQueueServer) {
	smqv1.RegisterSuhaibMessageQueueServer(registrar, server)
}
