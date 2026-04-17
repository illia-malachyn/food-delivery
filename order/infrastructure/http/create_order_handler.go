package http

import (
	"encoding/json"
	"net/http"

	"github.com/illia-malachyn/food-delivery/order/application"
)

func CreateOrderHandler(orderService *application.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var orderDTO application.OrderDTO
		err := json.NewDecoder(r.Body).Decode(&orderDTO)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = orderService.Create(r.Context(), &orderDTO)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
