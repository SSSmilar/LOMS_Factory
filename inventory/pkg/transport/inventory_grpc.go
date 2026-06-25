package transport

import (
	"context"
	"log/slog"
	"sort"

	"github.com/SSSmilar/LOMS_Factory/inventory/pkg/service"
	inventoryv1 "github.com/SSSmilar/LOMS_Factory/shared/pkg/proto/inventory/v1"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	reasonNotFound  = "not found"
	msgPartNotFound = "part not found"
)

// InventoryServer реализует gRPC сервис .
type InventoryServer struct {
	inventoryv1.UnimplementedInventoryServiceServer
	InventoryService *service.Service
}

// GetPart возвращает деталь по UUID .
func (s *InventoryServer) GetPart(
	ctx context.Context,
	req *inventoryv1.GetPartRequest,
) (*inventoryv1.GetPartResponse, error) {
	if req.GetUuid() == "" {
		sb := status.New(codes.InvalidArgument, "INVALID_ARGUMENT")

		w := &errdetails.BadRequest{
			FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{
					Field:       "uuid",
					Description: "uuid is required and cannot be empty",
				},
			},
		}
		sb, err := sb.WithDetails(w)
		if err != nil {
			return nil, status.Error(codes.Internal, "Internal error:")
		}
		slog.Warn("validation failed",
			slog.String("method", "GetPart"),
			slog.String("field", "uuid"),
			slog.String("reason", "empty"),
		)
		return nil, sb.Err()
	}
	number, err := uuid.Parse(req.GetUuid())
	if err != nil {
		sb := status.New(codes.InvalidArgument, "INVALID_ARGUMENT")
		w := &errdetails.BadRequest{
			FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{
					Field:       "uuid",
					Description: "invalid uuid format",
				},
			},
		}
		sb, err := sb.WithDetails(w)
		if err != nil {
			return nil, status.Error(codes.Internal, "Internal error:")
		}
		slog.Warn("validation failed",
			slog.String("method", "GetPart"),
			slog.String("field", "uuid"),
			slog.String("reason", "invalid formate"),
		)
		return nil, sb.Err()
	}
	part, err := s.InventoryService.GetPart(ctx, number.String())
	if err != nil {
		slog.Warn(msgPartNotFound,
			slog.String("method", "GetPart"),
			slog.String("field", "uuid"),
			slog.String("reason", reasonNotFound))
		return nil, status.Error(codes.NotFound, "NOT_FOUND")
	}

	return &inventoryv1.GetPartResponse{Part: mapPartToProto(part)}, nil
}

// ListParts возвращает список деталей с опциональной фильтрацией по типу .
func (s *InventoryServer) ListParts(
	ctx context.Context,
	req *inventoryv1.ListPartsRequest,
) (*inventoryv1.ListPartsResponse, error) {
	result := make([]*inventoryv1.Part, 0)
	if len(req.GetUuids()) > 0 {

		part, err := s.InventoryService.GetParts(ctx, req.GetUuids())
		if err != nil {
			slog.Warn(msgPartNotFound,
				slog.String("method", "ListParts"),
				slog.String("field", "uuid"),
				slog.String("reason", reasonNotFound))
			return nil, status.Error(codes.NotFound, "NOT_FOUND")
		}
		for _, p := range part {
			result = append(result, mapPartToProto(p))
		}
		return &inventoryv1.ListPartsResponse{Parts: result}, nil
	}

	targetType := req.GetPartType()
	parts, err := s.InventoryService.ListPartsByType(ctx, targetType)
	if err != nil {
		slog.Warn(msgPartNotFound,
			slog.String("method", "ListParts"),
			slog.String("field", "part_type"),
			slog.String("reason", reasonNotFound))
		return nil, status.Error(codes.NotFound, "NOT_FOUND")
	}
	for _, part := range parts {
		result = append(result, mapPartToProto(part))
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return &inventoryv1.ListPartsResponse{Parts: result}, nil
}
func mapPartToProto(part service.Part) *inventoryv1.Part {
	return &inventoryv1.Part{
		Uuid:          part.UUID,
		Name:          part.Name,
		Description:   part.Description,
		Price:         part.Price,
		PartType:      part.PartType,
		StockQuantity: part.StockQuantity,
		CreatedAt:     part.CreatedAt,
	}
}
