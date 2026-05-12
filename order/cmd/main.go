package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	orderHandler "github.com/SSSmilar/LOMS_Factory/order/pkg/handler"
	inventoryv1 "github.com/SSSmilar/LOMS_Factory/shared/pkg/proto/inventory/v1"
	paymentv1 "github.com/SSSmilar/LOMS_Factory/shared/pkg/proto/payment/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	inventoryServiceAddress = "localhost:50051"
	paymentServiceAddress   = "localhost:50052"
)

func main() {
	//conn, err := grpc.NewClient("localhost:50053",
	//	grpc.WithTransportCredentials(insecure.NewCredentials()),
	//	grpc.WithKeepaliveParams(keepalive.ClientParameters{
	//		Time:                10 * time.Second,
	//		Timeout:             3 * time.Second,
	//		PermitWithoutStream: true,
	//	}),
	//)
	//if err != nil {
	//	slog.Error("не удалось подключиться к inventory", "error", err)
	//	os.Exit(1)
	//}
	//defer conn.Close()
	// Создать gRPC соединение с InventoryService
	inventoryConn, err := grpc.NewClient(inventoryServiceAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("не удалось подключиться к InventoryService", "error", err)
		os.Exit(1)
	}
	defer inventoryConn.Close()

	paymentConn, err := grpc.NewClient(paymentServiceAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("не удалось подключиться к PaymentService", "error", err)
		os.Exit(1)
	}
	defer paymentConn.Close()

	// Создаём хранилище и обработчик
	store := orderHandler.NewOrderStore()
	h := orderHandler.NewOrderHandler(
		inventoryv1.NewInventoryServiceClient(inventoryConn),
		paymentv1.NewPaymentServiceClient(paymentConn),
		store,
	)

	orderServer, err := orderHandler.SetupServer(h)
	if err != nil {
		slog.Error("ошибка создания сервера OpenAPI", "error", err)
		os.Exit(1)
	}
	httpServer := &http.Server{
		Addr:              ":8080",
		Handler:           orderServer,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	go func() {
		slog.Info("🌐 HTTP Frontend запущен", " address", ":8080")
		if serveErr := httpServer.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			slog.Error("ошибка запуска HTTP сервера", "error", serveErr)
			cancel() // будим main, чтобы не висеть бесконечно
		}
	}()
	<-ctx.Done()
	slog.Info("🛑 остановка HTTP сервера")

	shutDownCtx, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelShutdown()
	if shutdownErr := httpServer.Shutdown(shutDownCtx); shutdownErr != nil {
		slog.Error("ошибка остановки HTTP сервера", "error", shutdownErr)
	}
	slog.Info("HTTP сервер остановлен")
}
