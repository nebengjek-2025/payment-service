package model

type DriverEvent struct {
	EventID      string       `json:"event_id"`
	OrderID      string       `json:"order_id"`
	PassengerID  string       `json:"passenger_id"`
	DriverID     string       `json:"driver_id"`
	RouteSummary RouteSummary `json:"route_summary,omitempty"`
}
