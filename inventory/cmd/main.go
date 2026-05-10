package main

import (
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	keepalive "google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	svc "github.com/SSSmilar/LOMS_Factory/inventory/pkg/service"
	inventoryv1 "github.com/SSSmilar/LOMS_Factory/shared/pkg/proto/inventory/v1"
)

const grpcAddress = ":50051"

func main() {
	lis, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		slog.Error("не удалось создать listener", "error", err)
		os.Exit(1)
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
	inventoryv1.RegisterInventoryServiceServer(grpcServer, svc.NewInventoryServer())

	// Включаем reflection для postman/grpcurl
	reflection.Register(grpcServer)

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
		os.Exit(1)
	}
}
