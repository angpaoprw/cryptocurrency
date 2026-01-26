package server

func (s *App) callbackRoute() {
	callback := s.Route.Group("/callback")
	{
		callback.Post("/alchemy", s.Controller.AlchemyCallback)
	}

}
