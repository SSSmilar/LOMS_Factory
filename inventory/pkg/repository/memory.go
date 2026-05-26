package repository

import (
	"context"
	"sync"

	"github.com/SSSmilar/LOMS_Factory/inventory/pkg/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type inMemoryRepo struct {
	parts map[string]service.Part
	mtx   sync.RWMutex
}

func (rep *inMemoryRepo) NewMemoryRepo() *inMemoryRepo {
	return &inMemoryRepo{
		parts: make(map[string]service.Part),
	}
}
func (rep *inMemoryRepo) GetParts(ctx context.Context, id string) (service.Part, error) {
	rep.mtx.RLock()
	defer rep.mtx.RUnlock()
	part, ok := rep.parts[id]
	if !ok {
		sb := status.New(codes.NotFound, "NOT_FOUND")
		return part, sb.Err()
	}
	return part, nil
}
func (rep *inMemoryRepo) ListParts(ctx context.Context, ids []uuid.UUID) ([]service.Part, error) {
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
