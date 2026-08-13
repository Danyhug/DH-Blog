package logging

import (
	"dh-blog/internal/middleware"
	"dh-blog/internal/utils"

	"github.com/sirupsen/logrus"
)

// ipService adapts middleware request records to the module's persistence
// model. Administrative ban operations stay inside the handler/repository.
type ipService struct {
	repository *Repository
	// resolveCity is injectable so tests can avoid real network calls.
	resolveCity func(ip string) (string, error)
}

func newIPService(repository *Repository) *ipService {
	return &ipService{repository: repository, resolveCity: utils.GetIPLocation}
}

func (s *ipService) RecordRequest(record middleware.AccessRecord) error {
	log := &AccessLog{
		IPAddress:    record.IPAddress,
		AccessDate:   record.AccessDate,
		UserAgent:    record.UserAgent,
		RequestURL:   record.RequestURL,
		ResourceType: record.ResourceType,
	}

	// City resolution goes through the repository's cache layers (memory →
	// ip_city_cache table → external API), so repeated visits never re-query
	// the external IP-location service.
	city, err := s.repository.ResolveCity(record.IPAddress, s.resolveCity)
	if err != nil {
		logrus.Warnf("获取IP地理位置信息失败: %v", err)
		city = "未知/未知"
	}
	if city == "本地网络" {
		city = "本地网络/本地/内网"
	}
	log.City = city

	return s.repository.SaveAccessLog(log)
}

func (s *ipService) IsIPBanned(ip string) (bool, error) {
	return s.repository.IsIPBanned(ip)
}

var _ middleware.IPService = (*ipService)(nil)
