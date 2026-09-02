package api

import (
	"net/http"
	"strconv"
)

func (a *API) listNotificationDeliveries(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0
	var err error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil {
			failure(w, 422, "Limit must be an integer", nil)
			return
		}
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil {
			failure(w, 422, "Offset must be an integer", nil)
			return
		}
	}
	history, err := a.store.ListNotificationDeliveryHistory(r.Context(), limit, offset)
	if err != nil {
		failure(w, 422, "Invalid delivery history request", err)
		return
	}
	write(w, 200, history)
}
