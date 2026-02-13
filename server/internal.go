package server

func (s *App) internalRoute() {
	internal := s.Route.Group("/internal")
	{
		internal.Post("/deposit", s.Controller.CreateInternalDepositRequest)
		internal.Delete("/deposit/:id", s.Controller.CancelDepositRequest)
		internal.Post("/withdrawal", s.Controller.CreateWithdrawalRequest)
		internal.Get("/withdrawal/:id", s.Controller.GetWithdrawalRequest)
	}
}
