package server

func (s *App) depositRoute() {
	deposit := s.Route.Group("/deposit")
	{
		deposit.Post("/request", s.Controller.CreateDepositRequest)
		deposit.Get("/request/:id", s.Controller.GetDepositRequestStatus)
	}
}
