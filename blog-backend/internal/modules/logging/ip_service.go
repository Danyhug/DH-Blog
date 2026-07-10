package logging

import "dh-blog/internal/middleware"

// ipService adapts middleware request records to the module's persistence
// model. Administrative ban operations stay inside the handler/repository.
type ipService struct {
	repository *Repository
}

func newIPService(repository *Repository) *ipService {
	return &ipService{repository: repository}
}

func (s *ipService) RecordRequest(record middleware.AccessRecord) error {
	return s.repository.SaveAccessLog(&AccessLog{
		IPAddress:    record.IPAddress,
		AccessDate:   record.AccessDate,
		UserAgent:    record.UserAgent,
		RequestURL:   record.RequestURL,
		City:         record.City,
		ResourceType: record.ResourceType,
	})
}

func (s *ipService) IsIPBanned(ip string) (bool, error) {
	return s.repository.IsIPBanned(ip)
}

var _ middleware.IPService = (*ipService)(nil)
