package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	paymentv1 "github.com/SSSmilar/LOMS_Factory/shared/pkg/proto/payment/v1"
)

// PaymentServer реализует gRPC сервис оплаты .
type PaymentServer struct {
	paymentv1.UnimplementedPaymentServiceServer
}

// PayOrder обрабатывает оплату заказа .
func (s *PaymentServer) PayOrder(
	ctx context.Context,
	req *paymentv1.PayOrderRequest,
) (*paymentv1.PayOrderResponse, error) {
	_ = ctx
	if len(req.GetOrderUuid()) == 0 {
		sb := status.New(codes.InvalidArgument, "INVALID_ARGUMENT")
		w := &errdetails.BadRequest{
			FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{
					Field:       "order_uuid",
					Description: "order_uuid is required and cannot be empty",
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
			slog.String("method", "PayOrder"),
			slog.String("field=order_uuid", req.GetOrderUuid()),
			slog.String("reason", "empty"),
		)
		return nil, sb.Err()
	}
	payMethod := req.GetPaymentMethod()
	if payMethod == paymentv1.PaymentMethod_PAYMENT_METHOD_UNSPECIFIED {
		sb := status.New(codes.InvalidArgument, "INVALID_ARGUMENT")
		w := &errdetails.BadRequest{
			FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{
					Field:       "Payment Method",
					Description: "Payment Method is Unspecified",
				},
			},
		}
		sb, err := sb.WithDetails(w)
		if err != nil {
			slog.Error("method PayOrder failed to attach error details:",
				slog.String("method", "ListParts"),
				slog.String("operation", "WithDetails"),
				slog.Any("error", err),
			)
			return nil, status.Errorf(codes.Internal, "internal error attaching details: %v", err)
		}
		slog.Warn("validation failed",
			slog.String("method", "PayOrder"),
			slog.String("field=PaymentMethod", payMethod.String()),
			slog.String("reason", "unspecified"),
		)
		return nil, sb.Err()
	}
	_, err := uuid.Parse(req.GetOrderUuid())
	if err != nil {
		sb := status.New(codes.InvalidArgument, "INVALID_ARGUMENT")
		w := &errdetails.BadRequest{
			FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{
					Field:       "order_uuid",
					Description: "invalid order_uuid format",
				},
			},
		}
		sb, err := sb.WithDetails(w)
		if err != nil {
			slog.Error("method PayOrder failed to attach error details:",
				slog.String("method", "ListParts"),
				slog.String("operation", "WithDetails"),
				slog.Any("error", err),
			)
			return nil, status.Errorf(codes.Internal, "internal error attaching details: %v", err)
		}
		slog.Warn("validation failed",
			slog.String("method", "PayOrder"),
			slog.String("field=order_uuid", req.GetOrderUuid()),
			slog.String("reason", "invalid format"),
		)
		return nil, sb.Err()
	}
	transactionUuid := uuid.New()

	slog.Info("оплата прошла успешно",
		"order_uuid", req.GetOrderUuid(),
		"transaction_uuid", transactionUuid,
	)
	return &paymentv1.PayOrderResponse{
		TransactionUuid: transactionUuid.String(),
	}, nil
}
