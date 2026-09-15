package models

type CampusOrder struct {
	ID               string `json:"id"`
	DestinationID    int    `json:"destinationId"`
	WeightGrams      int    `json:"weightGrams"`
	DeliveryFeeCents int    `json:"deliveryFeeCents"`
	ServiceSeconds   int    `json:"serviceSeconds"`
}

type BatchRequest struct {
	Count    int    `json:"count"`
	Seed     int64  `json:"seed"`
	Scenario string `json:"scenario"`
}

type OrderBatch struct {
	MapVersion string        `json:"mapVersion"`
	Seed       int64         `json:"seed"`
	Scenario   string        `json:"scenario"`
	Orders     []CampusOrder `json:"orders"`
}
