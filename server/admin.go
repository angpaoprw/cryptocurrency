package server

func (s *App) adminRoute() {
	admin := s.Route.Group("/admin")

	// Network information endpoints
	admin.Get("/networks", s.Controller.GetSupportedNetworks)
	admin.Get("/networks/:network_code/tokens", s.Controller.GetTokensByNetwork)

	// Token information endpoints
	admin.Get("/tokens", s.Controller.GetAllTokens)

	// Wallet management endpoints
	admin.Post("/wallets", s.Controller.CreateWallet)
	admin.Post("/wallets/import", s.Controller.ImportWallet)
	admin.Get("/wallets", s.Controller.ListWallets)
	admin.Get("/wallets/:id", s.Controller.GetWallet)
	admin.Put("/wallets/:id", s.Controller.UpdateWallet)

	// Webhook management endpoints
	admin.Get("/webhooks/status", s.Controller.WebhookStatus)
	admin.Post("/webhooks/bootstrap", s.Controller.BootstrapWebhooks)

	// Withdrawal management endpoints
	admin.Post("/withdrawal/:id/cancel", s.Controller.CancelWithdrawalRequest)
}
