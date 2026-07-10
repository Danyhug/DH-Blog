# Backend Modules

`internal/modules` uses vertical business modules. New features should use a
directory such as `modules/user` or
`modules/comment` instead of adding files to all of the global `controller`,
`service`, `repository`, and `model` packages.

## Module Shape

A normal module owns its HTTP routes, handler, business logic, persistence, and
database models:

```text
internal/modules/example/
  module.go       # dependency wiring, routes, exported module API
  handler.go      # HTTP input/output
  service.go      # use cases; omit when the feature has no business layer
  repository.go   # database access
  model.go        # module-owned database and DTO types
  module_test.go  # route and wiring contract
```

Keep a module in one Go package until its size or dependency graph justifies a
split. A directory per technical layer recreates the coupling this structure is
intended to remove.

The module entry point implements `router.Module`:

```go
type Module interface {
	RegisterRoutes(*router.Routes)
}
```

Constructors should assemble private dependencies inside the module:

```go
package example

type Module struct {
	repository *Repository
	handler    *Handler
}

func New(db *gorm.DB) *Module {
	repository := NewRepository(db)
	return &Module{
		repository: repository,
		handler:    NewHandler(repository),
	}
}

func (m *Module) RegisterRoutes(routes *router.Routes) {
	routes.PublicAPI.GET("/examples", m.handler.List)
	routes.AdminAPI.POST("/examples", m.handler.Create)
}

func MigrationModels() []any {
	return []any{&Example{}}
}
```

## Adding a Module

1. Create `internal/modules/<name>/` with the files the feature needs.
2. Keep construction inside `New`; do not make `app.New` assemble the module's
   repository, service, and handler one by one.
3. Add one entry to `moduleRegistrations` in `internal/app/registry.go`. The
   same entry supplies both route construction and `MigrationModels`, so there
   is no second schema list to maintain.
4. Add the module name to the expected order in `internal/app/registry_test.go`.
5. Add a route contract test and focused service/repository tests.

Use the route groups consistently:

- `PublicAPI`: public `/api` endpoints
- `AdminAPI`: JWT-protected `/api/admin` endpoints
- `AuthenticatedAPI("/api/files")`: JWT-protected legacy file endpoints
- `Engine`: routes outside those groups, such as public share links or WebDAV

Routes belong only in the module. Handlers must not expose a second
`RegisterRoutes` method.

## Cross-Module Dependencies

Expose only the narrow capability another module needs. Prefer an interface
defined by the consuming module over importing another module's concrete
handler or internal implementation. A module may temporarily expose a
repository or service during migration, but this should remain explicit at the
application composition root.

Shared infrastructure stays outside business modules, including `config`,
`database`, `dhcache`, `middleware`, `model` (shared persistence/pagination
types), `platform`, `response`, `router`, `task`, and `utils`.

## Migration Status

All current business features use vertical module directories: `admin`,
`article`, `comment`, `files`, `logging`, `share`, `system`, `user`, and
`webdav`. The old global `controller`, `service`, and `repository` layers have
been removed. New features should follow the same layout instead of recreating
those horizontal packages.
