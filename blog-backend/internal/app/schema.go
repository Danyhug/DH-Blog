package app

// SchemaModels lists the GORM models owned by the application.
func SchemaModels() []any {
	models := make([]any, 0, 16)
	for _, registration := range moduleRegistrations {
		models = append(models, registration.MigrationModels()...)
	}
	return models
}
