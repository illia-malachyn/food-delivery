package application

type OrderDTO struct {
	UserId   string `json:"user_id"`
	ItemId   string `json:"item_id"`
	Quantity uint   `json:"quantity"`
}
