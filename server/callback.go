package server

func (s *App) callbackRoute() {
	callback := s.App.Group("/callback")
	{
		callback.Get("/alchemy", s.Controller.AlchemyCallback)
	}

}
