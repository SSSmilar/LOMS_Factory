package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SSSmilar/LOMS_Factory/inventory/pkg/service"
	"github.com/SSSmilar/LOMS_Factory/inventory/pkg/transport"
	inventoryv1 "github.com/SSSmilar/LOMS_Factory/shared/pkg/proto/inventory/v1"
	"google.golang.org/grpc"
	keepalive "google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"github.com/SSSmilar/LOMS_Factory/inventory/pkg/repository" // Твоя база
)

const grpcAddress = ":50051"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	lc := net.ListenConfig{}
	lis, err := lc.Listen(ctx, "tcp", grpcAddress)
	if err != nil {
		slog.Error("failed to create listener", "error", err)
		return
	}
	kasp := keepalive.ServerParameters{
		MaxConnectionIdle:     15 * time.Minute,
		MaxConnectionAge:      1 * time.Hour,
		MaxConnectionAgeGrace: 30 * time.Second,
		Time:                  45 * time.Second,
		Timeout:               30 * time.Second,
	}
	kaep := keepalive.EnforcementPolicy{
		MinTime:             15 * time.Second,
		PermitWithoutStream: true,
	}

	grpcServer := grpc.NewServer(grpc.KeepaliveParams(kasp), grpc.KeepaliveEnforcementPolicy(kaep))

	repo := repository.NewMemoryRepo()

	svc := service.NewService(repo)
	// Включаем reflection для postman/grpcurl
	reflection.Register(grpcServer)

	inventoryServer := transport.InventoryServer{InventoryService: svc}

	inventoryv1.RegisterInventoryServiceServer(grpcServer, &inventoryServer)
	slog.Info("запуск InventoryService", "адрес", grpcAddress)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	go func() {
		<-quit
		slog.Info("остановка gRPC сервера")
		grpcServer.GracefulStop()
		slog.Info("сервер остановлен")
	}()

	err = grpcServer.Serve(lis)
	if err != nil {
		slog.Error("ошибка запуска сервера", "error", err)
	}
}
