# Backend Routes

`internal/modules` is the route-registration layer. It stays flat so the backend
does not grow a directory per feature. MVC code should live in the normal layers:

- `internal/model`: database and DTO structs
- `internal/controller`: HTTP request/response code
- `internal/service`: business logic
- `internal/repository`: database access

Each route module implements `router.Module`:

```go
type Module interface {
	RegisterRoutes(*router.Routes)
}
```

## Adding a Module

1. Create `internal/modules/<name>.go`.
2. Put the feature's route registration in `RegisterRoutes`.
3. Put new MVC code in `controller`, `service`, `repository`, and `model` as needed.
4. If the module owns database models, add them to `SchemaModels` in `internal/app/schema.go`.
5. Add the module to the slice in `internal/app/app.go`.

For example:

```go
package example

import "dh-blog/internal/router"

type Module struct {
	controller *Controller
}

func New(controller *Controller) *Module {
	return &Module{controller: controller}
}

func (m *Module) RegisterRoutes(routes *router.Routes) {
	routes.PublicAPI.GET("/example", m.controller.List)
	routes.AdminAPI.POST("/example", m.controller.Create)
}
```

Shared infrastructure still lives in packages such as `internal/config`, `internal/database`,
`internal/middleware`, `internal/response`, and `internal/utils`.
