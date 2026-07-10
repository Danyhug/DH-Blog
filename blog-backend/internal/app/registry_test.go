package app

import "testing"

func TestModuleRegistrationsAreCompleteAndPreserveRouteOrder(t *testing.T) {
	expectedOrder := []string{
		"article",
		"user",
		"comment",
		"admin",
		"logging",
		"system",
		"files",
		"share",
		"webdav",
	}
	if len(moduleRegistrations) != len(expectedOrder) {
		t.Fatalf("module registration count = %d, want %d", len(moduleRegistrations), len(expectedOrder))
	}

	seen := make(map[string]bool, len(moduleRegistrations))
	for i, registration := range moduleRegistrations {
		if registration.Name != expectedOrder[i] {
			t.Errorf("module registration %d = %q, want %q", i, registration.Name, expectedOrder[i])
		}
		if registration.Name == "" {
			t.Errorf("module registration %d has no name", i)
		}
		if seen[registration.Name] {
			t.Errorf("module %q is registered more than once", registration.Name)
		}
		seen[registration.Name] = true
		if registration.MigrationModels == nil {
			t.Errorf("module %q does not declare migration models", registration.Name)
		}
		if registration.Build == nil {
			t.Errorf("module %q does not declare a build function", registration.Name)
		}
	}
}
