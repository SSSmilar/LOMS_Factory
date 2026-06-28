package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/SSSmilar/LOMS_Factory/inventory/pkg/service"
	inventoryv1 "github.com/SSSmilar/LOMS_Factory/shared/pkg/proto/inventory/v1"
)

type inMemoryRepo struct {
	parts map[string]service.Part
	mtx   sync.RWMutex
}

func NewMemoryRepo() *inMemoryRepo {
	return &inMemoryRepo{
		parts: make(map[string]service.Part),
	}
}
func (rep *inMemoryRepo) GetPart(ctx context.Context, id string) (service.Part, error) {
	rep.mtx.RLock()
	defer rep.mtx.RUnlock()
	part, ok := rep.parts[id]
	if !ok {
		return part, fmt.Errorf("part not found in method GetPart")
	}
	return part, nil
}
func (rep *inMemoryRepo) GetParts(ctx context.Context, ids []string) ([]service.Part, error) {
	parts := make([]service.Part, 0, len(ids))

	rep.mtx.RLock()
	defer rep.mtx.RUnlock()

	for _, id := range ids {
		part, ok := rep.parts[id]
		if !ok {
			return parts, fmt.Errorf("part not found")
		}
		parts = append(parts, part)
	}
	return parts, nil
}
func (rep *inMemoryRepo) ListPartsByType(ctx context.Context, targetType inventoryv1.PartType) ([]service.Part, error) {
	var result []service.Part
	rep.mtx.RLock()
	defer rep.mtx.RUnlock()
	for _, partItem := range rep.parts {
		if partItem.PartType == targetType {
			result = append(result, partItem)
		}
	}
	return result, nil
}
