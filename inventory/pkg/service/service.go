package service

import (
	"context"
	"log/slog"
	"sort"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	inventoryv1 "github.com/SSSmilar/LOMS_Factory/shared/pkg/proto/inventory/v1"
)

// Part представляет деталь космического корабля .
type Part struct {
	UUID          string
	Name          string
	Description   string
	Price         int64 // в копейках
	PartType      inventoryv1.PartType
	StockQuantity int64
	CreatedAt     *timestamppb.Timestamp
}

// InventoryServer реализует gRPC сервис .
type InventoryServer struct {
	inventoryv1.UnimplementedInventoryServiceServer
	parts map[uuid.UUID]Part
}

// NewInventoryServer создаёт сервер с предзагруженными seed-данными .
func NewInventoryServer() *InventoryServer {
	now := timestamppb.Now()

	return &InventoryServer{
		parts: map[uuid.UUID]Part{
			uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"): {
				UUID:          "550e8400-e29b-41d4-a716-446655440001",
				Name:          "Алюминиевый корпус",
				Description:   "Лёгкий корпус для небольших кораблей",
				Price:         500000, // 5000₽
				PartType:      inventoryv1.PartType_PART_TYPE_HULL,
				StockQuantity: 10,
				CreatedAt:     now,
			},
			uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"): {
				UUID:          "550e8400-e29b-41d4-a716-446655440002",
				Name:          "Титановый корпус",
				Description:   "Прочный корпус для средних кораблей",
				Price:         1500000, // 15000₽
				PartType:      inventoryv1.PartType_PART_TYPE_HULL,
				StockQuantity: 5,
				CreatedAt:     now,
			},
			uuid.MustParse("550e8400-e29b-41d4-a716-446655440003"): {
				UUID:          "550e8400-e29b-41d4-a716-446655440003",
				Name:          "Ионный двигатель C",
				Description:   "Базовый ионный двигатель класса C",
				Price:         300000, // 3000₽
				PartType:      inventoryv1.PartType_PART_TYPE_ENGINE,
				StockQuantity: 8,
				CreatedAt:     now,
			},
			uuid.MustParse("550e8400-e29b-41d4-a716-446655440004"): {
				UUID:          "550e8400-e29b-41d4-a716-446655440004",
				Name:          "Ионный двигатель B",
				Description:   "Улучшенный ионный двигатель класса B",
				Price:         800000, // 8000₽
				PartType:      inventoryv1.PartType_PART_TYPE_ENGINE,
				StockQuantity: 3,
				CreatedAt:     now,
			},
			uuid.MustParse("550e8400-e29b-41d4-a716-446655440005"): {
				UUID:          "550e8400-e29b-41d4-a716-446655440005",
				Name:          "Энергетический щит",
				Description:   "Стандартный энергетический щит",
				Price:         400000, // 4000₽
				PartType:      inventoryv1.PartType_PART_TYPE_SHIELD,
				StockQuantity: 6,
				CreatedAt:     now,
			},
			uuid.MustParse("550e8400-e29b-41d4-a716-446655440006"): {
				UUID:          "550e8400-e29b-41d4-a716-446655440006",
				Name:          "Лазерная пушка",
				Description:   "Точная лазерная пушка",
				Price:         250000, // 2500₽
				PartType:      inventoryv1.PartType_PART_TYPE_WEAPON,
				StockQuantity: 7,
				CreatedAt:     now,
			},
			uuid.MustParse("550e8400-e29b-41d4-a716-446655440007"): {
				UUID:          "550e8400-e29b-41d4-a716-446655440007",
				Name:          "Плазменный корпус",
				Description:   "Экспериментальный корпус (нет на складе)",
				Price:         2000000, // 20000₽
				PartType:      inventoryv1.PartType_PART_TYPE_HULL,
				StockQuantity: 0,
				CreatedAt:     now,
			},
		},
	}
}
func mapPartToProto(part Part) *inventoryv1.Part {
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
	part, ok := s.parts[number]
	if !ok {
		slog.Warn("part not found",
			slog.String("method", "GetPart"),
			slog.String("field", "uuid"),
			slog.String("reason", "not found"))
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
		for _, i := range req.GetUuids() {
			idStr, err := uuid.Parse(i)
			if err != nil {
				sb := status.New(codes.InvalidArgument, "INVALID_ARGUMENT")
				w := &errdetails.BadRequest{
					FieldViolations: []*errdetails.BadRequest_FieldViolation{
						{
							Field:       "uuids",
							Description: "invalid uuid: they hand over invalid format",
						},
					},
				}
				sb, err := sb.WithDetails(w)
				if err != nil {
					slog.Error("failed to attach error details",
						slog.String("method", "ListParts"),
						slog.String("operation", "WithDetails"),
						slog.Any("error", err),
					)
					return nil, status.Errorf(codes.Internal, "internal error attaching details: %v", err)
				}
				slog.Warn("validation failed",
					slog.String("method", "ListParts"),
					slog.String("field", "uuid"),
					slog.String("reason", "invalid format"),
				)
				return nil, sb.Err()
			}
			part, ok := s.parts[idStr]
			if !ok {
				slog.Warn("part not found",
					slog.String("method", "ListParts"),
					slog.String("field", "uuid"),
					slog.String("reason", "not found"))
				return nil, status.Error(codes.NotFound, "NOT_FOUND")
			}
			result = append(result, mapPartToProto(part))
		}
		return &inventoryv1.ListPartsResponse{Parts: result}, nil
	}
	for _, part := range s.parts {
		targetType := req.GetPartType()
		if targetType == inventoryv1.PartType_PART_TYPE_UNSPECIFIED || part.PartType == targetType {
			result = append(result, mapPartToProto(part))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return &inventoryv1.ListPartsResponse{Parts: result}, nil
}
