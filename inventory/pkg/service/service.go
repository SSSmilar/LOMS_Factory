package service

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/protobuf/types/known/timestamppb"

	inventoryv1 "github.com/SSSmilar/LOMS_Factory/shared/pkg/proto/inventory/v1"
)

type InventoryRepository interface {
	GetParts(ctx context.Context, uuids []string) ([]Part, error)
	ListPartsByType(ctx context.Context, partType inventoryv1.PartType) ([]Part, error)
	GetPart(ctx context.Context, uuid string) (Part, error)
}

type Service struct {
	inventoryRepository InventoryRepository
}

func NewService(inventoryRepository InventoryRepository) *Service {
	return &Service{
		inventoryRepository: inventoryRepository,
	}
}

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

func (s *Service) ListPartsByType(ctx context.Context, targetType inventoryv1.PartType) ([]Part, error) {
	result, err := s.inventoryRepository.ListPartsByType(ctx, targetType)
	if err != nil {
		slog.Warn("failed to get parts by type",
			slog.String("method", "ListPartsByType"),
			slog.String("reason", "internal error"),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to get parts by type: %w", err)
	}
	return result, nil
}
func (s *Service) GetParts(ctx context.Context, uuid []string) ([]Part, error) {
	return s.inventoryRepository.GetParts(ctx, uuid)
}

func (s *Service) GetPart(ctx context.Context, uuid string) (Part, error) {
	return s.inventoryRepository.GetPart(ctx, uuid)
}
