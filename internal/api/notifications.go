package api

func (a *API) registerNotificationRoutes() {
	a.mux.HandleFunc("POST /api/v1/notifications/test", a.testNotification)
	a.mux.HandleFunc("GET /api/v1/notifications", a.listNotificationDestinations)
	a.mux.HandleFunc("POST /api/v1/notifications", a.createNotificationDestination)
	a.mux.HandleFunc("PUT /api/v1/notifications/{id}", a.updateNotificationDestination)
	a.mux.HandleFunc("DELETE /api/v1/notifications/{id}", a.deleteNotificationDestination)
	a.mux.HandleFunc("GET /api/v1/notification-routes", a.listNotificationRoutes)
	a.mux.HandleFunc("POST /api/v1/notification-routes", a.createNotificationRoute)
	a.mux.HandleFunc("PUT /api/v1/notification-routes/{id}", a.updateNotificationRoute)
	a.mux.HandleFunc("DELETE /api/v1/notification-routes/{id}", a.deleteNotificationRoute)
	a.mux.HandleFunc("GET /api/v1/notification-deliveries", a.listNotificationDeliveries)
}
