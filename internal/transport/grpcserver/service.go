package grpcserver

import (
	"context"
	"errors"

	miniragv1 "github.com/pfyao1101/miniRAG/api/minirag/v1"
	"github.com/pfyao1101/miniRAG/internal/store"
	"github.com/pfyao1101/miniRAG/internal/vector"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	// 生成器要求嵌入
	miniragv1.UnimplementedVectorStoreServiceServer

	// 一个接口
	// memoryStore 实现了这个接口，将这个变量设置为接口是方便后续进行 Store 的替换
	backend store.VectorStore
}

func NewService(backend store.VectorStore) *Service {
	return &Service{
		backend: backend,
	}
}

func (service *Service) Insert(
	ctx context.Context,
	req *miniragv1.InsertRequest,
) (*miniragv1.InsertResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, toStatusError(err)
	}

	// Getter 是 nil-safe 的，因此这里需要显式检查 Record 是否存在。
	if req == nil || req.GetRecord() == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"record is required",
		)
	}

	reqRecord := req.GetRecord()
	record := store.Record{
		ID:       reqRecord.GetId(),
		Vector:   reqRecord.GetVector(),
		Text:     reqRecord.GetText(),
		Metadata: reqRecord.GetMetadata(),
	}

	if err := service.backend.Insert(ctx, record); err != nil {
		return nil, toStatusError(err)
	}

	return &miniragv1.InsertResponse{}, nil
}

func (service *Service) Get(ctx context.Context, req *miniragv1.GetRequest) (*miniragv1.GetResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, toStatusError(err)
	}

	if req == nil || req.GetId() == "" {
		return nil, status.Error(
			codes.InvalidArgument,
			"id is required",
		)
	}

	id := req.GetId()
	record, err := service.backend.Get(ctx, id)
	if err != nil {
		return nil, toStatusError(err)
	}

	return &miniragv1.GetResponse{
		Record: toProtoRecord(record),
	}, nil
}

func (service *Service) Delete(ctx context.Context, req *miniragv1.DeleteRequest) (*miniragv1.DeleteResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, toStatusError(err)
	}

	if req == nil || req.GetId() == "" {
		return nil, status.Error(
			codes.InvalidArgument,
			"id is required",
		)
	}

	id := req.GetId()

	if err := service.backend.Delete(ctx, id); err != nil {
		return nil, toStatusError(err)
	}

	return &miniragv1.DeleteResponse{}, nil
}

func (service *Service) Search(ctx context.Context, req *miniragv1.SearchRequest) (*miniragv1.SearchResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, toStatusError(err)
	}

	// Getter 是 nil-safe 的，因此这里需要显式检查 Query 是否存在。
	if req == nil || req.GetQuery() == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"query is required",
		)
	}

	reqQuery := req.GetQuery()
	reqK := req.GetK()

	results, err := service.backend.Search(ctx, reqQuery, int(reqK))
	if err != nil {
		return nil, toStatusError(err)
	}

	responseResults := make([]*miniragv1.SearchResult, len(results))
	for i, result := range results {
		responseResults[i] = &miniragv1.SearchResult{
			Id:    result.ID,
			Score: result.Score,
		}
	}

	return &miniragv1.SearchResponse{
		Results: responseResults,
	}, nil
}

// 编译器接口检查，如果 Service 没有满足生成的服务端接口，编译会失败。
var _ miniragv1.VectorStoreServiceServer = (*Service)(nil)

func toStatusError(err error) error {
	switch {
	case err == nil:
		return nil

	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, err.Error())

	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, err.Error())

	case errors.Is(err, store.ErrEmptyID),
		errors.Is(err, store.ErrVectorDimensionMismatch),
		errors.Is(err, vector.ErrEmptyVector),
		errors.Is(err, vector.ErrZeroVector),
		errors.Is(err, vector.ErrInvalidK):
		return status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, store.ErrDuplicateID):
		return status.Error(codes.AlreadyExists, err.Error())

	case errors.Is(err, store.ErrRecordNotFound):
		return status.Error(codes.NotFound, err.Error())

	default:
		return status.Error(
			codes.Internal,
			"internal server error",
		)
	}
}

func toProtoRecord(record store.Record) *miniragv1.Record {
	return &miniragv1.Record{
		Id:       record.ID,
		Vector:   record.Vector,
		Text:     record.Text,
		Metadata: record.Metadata,
	}
}
