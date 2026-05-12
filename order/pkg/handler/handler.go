package handler

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	orderv1 "github.com/SSSmilar/LOMS_Factory/shared/pkg/openapi/order/v1"
	inventoryv1 "github.com/SSSmilar/LOMS_Factory/shared/pkg/proto/inventory/v1"
	paymentv1 "github.com/SSSmilar/LOMS_Factory/shared/pkg/proto/payment/v1"
)

// Order представляет заказ на постройку космического корабля.
type Order struct {
	OrderUUID       uuid.UUID
	HullUUID        uuid.UUID
	EngineUUID      uuid.UUID
	ShieldUUID      *uuid.UUID // опциональный
	WeaponUUID      *uuid.UUID // опциональный
	TotalPrice      int64      // в копейках
	TransactionUUID *uuid.UUID
	PaymentMethod   *string
	Status          string // PENDING_PAYMENT, PAID, CANCELLED
	CreatedAt       time.Time
}

// OrderStore — хранилище заказов (in-memory).
type OrderStore struct {
	mu     sync.RWMutex
	orders map[uuid.UUID]Order
}

// NewOrderStore создаёт новое пустое хранилище заказов.
func NewOrderStore() *OrderStore {
	return &OrderStore{
		orders: make(map[uuid.UUID]Order),
	}
}

// OrderHandler реализует интерфейс orderv1.Handler, сгенерированный ogen.
type OrderHandler struct {
	orderv1.UnimplementedHandler
	inventoryClient inventoryv1.InventoryServiceClient
	paymentClient   paymentv1.PaymentServiceClient
	store           *OrderStore
}

// NewOrderHandler создаёт новый обработчик заказов.
func NewOrderHandler(
	inventoryClient inventoryv1.InventoryServiceClient,
	paymentClient paymentv1.PaymentServiceClient,
	store *OrderStore,
) *OrderHandler {
	return &OrderHandler{
		inventoryClient: inventoryClient,
		paymentClient:   paymentClient,
		store:           store,
	}
}

// SetupServer создаёт OpenAPI сервер на основе обработчика.
func SetupServer(h *OrderHandler) (*orderv1.Server, error) {
	return orderv1.NewServer(h)
}

func (h *OrderHandler) fetchAndValidatePart(
	ctx context.Context,
	partType inventoryv1.PartType,
	partUUID string,
) (*inventoryv1.Part, orderv1.CreateOrderRes) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &inventoryv1.ListPartsRequest{
		PartType: partType,
		Uuids:    []string{partUUID},
	}

	resp, err := h.inventoryClient.ListParts(ctxWithTimeout, req)
	if err != nil {
		return nil, &orderv1.CreateOrderInternalServerError{
			Code:    http.StatusInternalServerError,
			Message: "inventory service error",
		}
	}

	if len(resp.GetParts()) == 0 {
		return nil, &orderv1.CreateOrderNotFound{
			Code:    http.StatusNotFound,
			Message: "part not found",
		}
	}

	part := resp.GetParts()[0]

	if part.StockQuantity == 0 {
		return nil, &orderv1.CreateOrderConflict{
			Code:    http.StatusConflict,
			Message: "part out of stock",
		}
	}

	return part, nil
}

// GetOrder реализует операцию getOrder (пример реализации).
// GET /api/v1/orders/{order_uuid}.
func (h *OrderHandler) GetOrder(_ context.Context, params orderv1.GetOrderParams) (orderv1.GetOrderRes, error) {
	// 1. Найти заказ в store (с блокировкой для thread-safety)
	h.store.mu.RLock()
	order, ok := h.store.orders[params.OrderUUID]
	h.store.mu.RUnlock()

	// 2. Если не найден — вернуть 404
	if !ok {
		return &orderv1.GetOrderNotFound{
			Code:    http.StatusNotFound,
			Message: "заказ не найден",
		}, nil
	}

	// 3. Преобразовать в DTO и вернуть
	var shieldUUID orderv1.OptNilUUID
	if order.ShieldUUID != nil {
		shieldUUID = orderv1.NewOptNilUUID(*order.ShieldUUID)
	}

	var weaponUUID orderv1.OptNilUUID
	if order.WeaponUUID != nil {
		weaponUUID = orderv1.NewOptNilUUID(*order.WeaponUUID)
	}

	var transactionUUID orderv1.OptNilUUID
	if order.TransactionUUID != nil {
		transactionUUID = orderv1.NewOptNilUUID(*order.TransactionUUID)
	}

	var paymentMethod orderv1.OptNilPaymentMethod
	if order.PaymentMethod != nil {
		paymentMethod = orderv1.NewOptNilPaymentMethod(orderv1.PaymentMethod(*order.PaymentMethod))
	}

	return &orderv1.OrderDto{
		OrderUUID:       order.OrderUUID,
		HullUUID:        order.HullUUID,
		EngineUUID:      order.EngineUUID,
		ShieldUUID:      shieldUUID,
		WeaponUUID:      weaponUUID,
		TotalPrice:      order.TotalPrice,
		TransactionUUID: transactionUUID,
		PaymentMethod:   paymentMethod,
		Status:          orderv1.OrderStatus(order.Status),
		CreatedAt:       order.CreatedAt,
	}, nil
}

func (h *OrderHandler) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (orderv1.CreateOrderRes, error) {
	var totalPrice int64
	ctxWithTimeoutHull, cancelHull := context.WithTimeout(ctx, 3*time.Second)
	defer cancelHull()

	if uuid.Nil == req.HullUUID {
		return &orderv1.CreateOrderBadRequest{
			Code:    http.StatusBadRequest,
			Message: "HullUuid is required and cannot be empty.",
		}, nil
	}
	hull, err := h.fetchAndValidatePart(ctxWithTimeoutHull, inventoryv1.PartType_PART_TYPE_HULL, req.HullUUID.String())
	if err != nil {
		return err, nil
	}
	totalPrice += hull.Price

	ctxWithTimeoutEngine, cancelEngine := context.WithTimeout(ctx, 3*time.Second)
	defer cancelEngine()
	if uuid.Nil == req.EngineUUID {
		return &orderv1.CreateOrderBadRequest{
			Code:    http.StatusBadRequest,
			Message: "EngineUuid is required and cannot be empty.",
		}, nil
	}

	engine, err := h.fetchAndValidatePart(ctxWithTimeoutEngine, inventoryv1.PartType_PART_TYPE_ENGINE, req.EngineUUID.String())
	if err != nil {
		return err, nil
	}
	totalPrice += engine.Price

	var shieldUuidId *uuid.UUID
	if req.ShieldUUID.Set && !req.ShieldUUID.Null {
		ctxWithTimeoutShield, cancelShield := context.WithTimeout(ctx, 3*time.Second)
		defer cancelShield()
		shieldUuidId = &req.ShieldUUID.Value
		shield, err := h.fetchAndValidatePart(ctxWithTimeoutShield, inventoryv1.PartType_PART_TYPE_SHIELD, shieldUuidId.String())
		if err != nil {
			return err, nil
		}
		totalPrice += shield.Price
	}

	var weaponUUID *uuid.UUID
	if req.WeaponUUID.Set && !req.WeaponUUID.Null {
		ctxWithTimeoutWeapon, cancelWeapon := context.WithTimeout(ctx, 3*time.Second)
		defer cancelWeapon()
		weaponUUID = &req.WeaponUUID.Value
		weapon, err := h.fetchAndValidatePart(ctxWithTimeoutWeapon, inventoryv1.PartType_PART_TYPE_WEAPON, weaponUUID.String())
		if err != nil {
			return err, nil
		}
		totalPrice += weapon.Price
	}
	orderUuid := uuid.New()

	myOrder := Order{
		OrderUUID:       orderUuid,
		HullUUID:        req.HullUUID,
		EngineUUID:      req.EngineUUID,
		ShieldUUID:      shieldUuidId,
		WeaponUUID:      weaponUUID,
		TotalPrice:      totalPrice,
		TransactionUUID: nil,
		PaymentMethod:   nil,
		Status:          "PENDING_PAYMENT",
		CreatedAt:       time.Now(),
	}
	h.store.mu.Lock()
	defer h.store.mu.Unlock()
	h.store.orders[myOrder.OrderUUID] = myOrder

	return &orderv1.CreateOrderResponse{
		OrderUUID:  orderUuid,
		TotalPrice: totalPrice,
	}, nil
}

// PayOrder реализует операцию payOrder.
// POST /api/v1/orders/{order_uuid}/pay.
func (h *OrderHandler) PayOrder(ctx context.Context, req *orderv1.PayOrderRequest, params orderv1.PayOrderParams) (orderv1.PayOrderRes, error) {
	h.store.mu.RLock()
	order, ok := h.store.orders[params.OrderUUID]
	if !ok {
		return &orderv1.PayOrderNotFound{
			Code:    http.StatusNotFound,
			Message: "order not found",
		}, nil
	}
	if order.Status != "PENDING_PAYMENT" {
		return &orderv1.PayOrderConflict{
			Code:    http.StatusConflict,
			Message: "status is not pending payment",
		}, nil
	}
	h.store.mu.RUnlock()
	var paymentMethod paymentv1.PaymentMethod
	switch req.GetPaymentMethod() {
	case orderv1.PaymentMethod("CARD"):
		paymentMethod = paymentv1.PaymentMethod_PAYMENT_METHOD_CARD
	case orderv1.PaymentMethod("SBP"):
		paymentMethod = paymentv1.PaymentMethod_PAYMENT_METHOD_SBP
	case orderv1.PaymentMethod("CREDIT_CARD"):
		paymentMethod = paymentv1.PaymentMethod_PAYMENT_METHOD_CREDIT_CARD
	case orderv1.PaymentMethod("INVESTOR_MONEY"):
		paymentMethod = paymentv1.PaymentMethod_PAYMENT_METHOD_INVESTOR_MONEY
	default:
		return nil, errors.New("unknown payment method")
	}
	payReq := &paymentv1.PayOrderRequest{
		OrderUuid:     order.OrderUUID.String(),
		PaymentMethod: paymentMethod,
	}
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	payRes, err := h.paymentClient.PayOrder(ctxWithTimeout, payReq)
	if err != nil {
		return &orderv1.PayOrderInternalServerError{
			Code:    http.StatusInternalServerError,
			Message: "payment error: " + err.Error() + "",
		}, nil
	}
	h.store.mu.Lock()
	defer h.store.mu.Unlock()
	order = h.store.orders[params.OrderUUID]
	order.Status = "PAID"
	transactionUuid, err := uuid.Parse(payRes.GetTransactionUuid())
	if err != nil {
		return &orderv1.PayOrderInternalServerError{
			Code:    http.StatusInternalServerError,
			Message: "order server error",
		}, nil
	}
	order.TransactionUUID = &transactionUuid
	h.store.orders[order.OrderUUID] = order
	return &orderv1.PayOrderResponse{TransactionUUID: transactionUuid}, nil
}

// CancelOrder реализует операцию cancelOrder
// POST /api/v1/orders/{order_uuid}/cancel.
func (h *OrderHandler) CancelOrder(_ context.Context, params orderv1.CancelOrderParams) (orderv1.CancelOrderRes, error) {
	h.store.mu.RLock()
	order, ok := h.store.orders[params.OrderUUID]
	if !ok {
		return &orderv1.CancelOrderNotFound{
			Code:    http.StatusNotFound,
			Message: "order not found",
		}, nil
	}
	if order.Status != "PENDING_PAYMENT" {
		return &orderv1.CancelOrderConflict{
			Code:    http.StatusConflict,
			Message: "status is not pending payment",
		}, nil
	}
	h.store.mu.RUnlock()
	h.store.mu.Lock()
	defer h.store.mu.Unlock()
	order = h.store.orders[params.OrderUUID]
	order.Status = "CANCELLED"
	h.store.orders[order.OrderUUID] = order
	return &orderv1.CancelOrderResponse{}, nil
}
