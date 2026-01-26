package server

func (s *App) callbackRoute() {
	callback := s.App.Group("/callback")
	{
		callback.Post("/alchemy", s.Controller.AlchemyCallback)
	}

}
