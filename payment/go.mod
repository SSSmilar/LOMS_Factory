
module github.com/SSSmilar/LOMS_Factory/payment

go 1.26.0

require (
	github.com/SSSmilar/LOMS_Factory/shared v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.79.3
)

require (
	go.opentelemetry.io/otel/sdk/metric v1.42.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/SSSmilar/LOMS_Factory/shared => ./../shared
