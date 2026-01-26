# Swagger Documentation

This directory contains auto-generated Swagger/OpenAPI documentation for the Cryptocurrency API.

## Accessing Swagger UI

Once the server is running, you can access the Swagger UI at:

```
http://localhost:5000/swagger/index.html
```

## Regenerating Documentation

After making changes to API annotations in the code, regenerate the documentation by running:

```bash
swag init
```

This will update:

- `docs/docs.go`
- `docs/swagger.json`
- `docs/swagger.yaml`

## Adding Swagger Annotations

To document your API endpoints, add annotations above your handler functions:

```go
// HandlerName godoc
// @Summary Brief description
// @Description Detailed description
// @Tags TagName
// @Accept json
// @Produce json
// @Param id path string true "ID"
// @Success 200 {object} ResponseType
// @Failure 400 {object} ErrorType
// @Router /endpoint/{id} [get]
func HandlerName(c *fiber.Ctx) error {
    // handler code
}
```

## General API Info

The general API information is defined in `main.go`:

```go
// @title Cryptocurrency API
// @version 1.0
// @description API description
// @host localhost:5000
// @BasePath /v1
```
