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
		return nil, errors.New("hull_uuid is required")
	}
	hullReq := &inventoryv1.ListPartsRequest{
		PartType: inventoryv1.PartType_PART_TYPE_HULL,
		Uuids:    []string{req.HullUUID.String()},
	}

	hullResp, err := h.inventoryClient.ListParts(ctxWithTimeoutHull, hullReq)
	if err != nil {
		return nil, errors.New("Response error: " + err.Error())
	}

	if len(hullResp.GetParts()) <= 0 {
		return nil, errors.New("Hull not found")
	}

	hull := hullResp.GetParts()[0]

	if hull.StockQuantity <= 0 {
		return nil, errors.New("Hull out of stock")
	}
	totalPrice += hull.Price

	ctxWithTimeoutEngine, cancelEngine := context.WithTimeout(ctx, 3*time.Second)
	defer cancelEngine()
	if uuid.Nil == req.EngineUUID {
		return nil, errors.New("engine_uuid is required")
	}

	engineReq := &inventoryv1.ListPartsRequest{
		PartType: inventoryv1.PartType_PART_TYPE_ENGINE,
		Uuids:    []string{req.EngineUUID.String()},
	}

	engineResp, err := h.inventoryClient.ListParts(ctxWithTimeoutEngine, engineReq)
	if err != nil {
		return nil, errors.New("Response error: " + err.Error())
	}

	if len(engineResp.GetParts()) <= 0 {
		return nil, errors.New("Engine not found")
	}

	engine := engineResp.GetParts()[0]

	if engine.StockQuantity <= 0 {
		return nil, errors.New("Engine out of stock")
	}
	totalPrice += engine.Price

	var shieldUuidId *uuid.UUID
	if req.ShieldUUID.Set && !req.ShieldUUID.Null {
		ctxWithTimeoutShield, cancelShield := context.WithTimeout(ctx, 3*time.Second)
		defer cancelShield()
		shieldUuidId = &req.ShieldUUID.Value
		shieldReq := &inventoryv1.ListPartsRequest{
			PartType: inventoryv1.PartType_PART_TYPE_SHIELD,
			Uuids:    []string{shieldUuidId.String()},
		}

		shieldResp, err := h.inventoryClient.ListParts(ctxWithTimeoutShield, shieldReq)
		if err != nil {
			return nil, errors.New("Response error: " + err.Error())
		}

		if len(shieldResp.GetParts()) <= 0 {
			return nil, errors.New("Shield not found")
		}

		shield := shieldResp.GetParts()[0]

		if shield.StockQuantity <= 0 {
			return nil, errors.New("Shield out of stock")
		}
		totalPrice += shield.Price
	}

	var weaponUUID *uuid.UUID
	if req.WeaponUUID.Set && !req.WeaponUUID.Null {
		ctxWithTimeoutWeapon, cancelWeapon := context.WithTimeout(ctx, 3*time.Second)
		defer cancelWeapon()
		weaponUUID = &req.WeaponUUID.Value

		weaponReq := &inventoryv1.ListPartsRequest{
			PartType: inventoryv1.PartType_PART_TYPE_WEAPON,
			Uuids:    []string{weaponUUID.String()},
		}

		weaponResp, err := h.inventoryClient.ListParts(ctxWithTimeoutWeapon, weaponReq)
		if err != nil {
			return nil, errors.New("Response error: " + err.Error())
		}

		if len(weaponResp.GetParts()) <= 0 {
			return nil, errors.New("Weapon not found")
		}

		weapon := weaponResp.GetParts()[0]

		if weapon.StockQuantity <= 0 {
			return nil, errors.New("Weapon out of stock")
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

//
// PayOrder реализует операцию payOrder
// POST /api/v1/orders/{order_uuid}/pay
// func (h *OrderHandler) PayOrder(ctx context.Context, req *orderv1.PayOrderRequest, params orderv1.PayOrderParams) (orderv1.PayOrderRes, error) {
//     // 1. Найти заказ в store
//     // 2. Проверить статус == PENDING_PAYMENT
//     // 3. Вызвать h.paymentClient.PayOrder для обработки платежа
//     // 4. Обновить статус на PAID и сохранить transaction_uuid
//     // 5. Вернуть transaction_uuid
// }
//
// CancelOrder реализует операцию cancelOrder
// POST /api/v1/orders/{order_uuid}/cancel
// func (h *OrderHandler) CancelOrder(ctx context.Context, params orderv1.CancelOrderParams) (orderv1.CancelOrderRes, error) {
//     // 1. Найти заказ в store
//     // 2. Проверить статус == PENDING_PAYMENT
//     // 3. Обновить статус на CANCELLED
//     // 4. Вернуть success
// }
