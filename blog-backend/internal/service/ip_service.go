package service

import (
	"time"

	"dh-blog/internal/model"
	"dh-blog/internal/repository"
	"github.com/sirupsen/logrus"
)

type IPService interface {
	// RecordRequest 记录请求
	RecordRequest(log *model.AccessLog) error
	// IsIPBanned 查看IP是否被封禁
	IsIPBanned(ip string) (bool, error)
	// BanIP 封禁IP
	BanIP(ip, reason string, expireTime time.Time) error
	// UnbanIP 解封IP
	UnbanIP(ip string) error
}

type ipService struct {
	repo *repository.LogRepository
}

func NewIPService(logRepo *repository.LogRepository) IPService {
	return &ipService{
		repo: logRepo,
	}
}

func (i *ipService) RecordRequest(log *model.AccessLog) error {
	return i.repo.SaveAccessLog(log)
}

func (i *ipService) IsIPBanned(ip string) (bool, error) {
	return i.repo.IsIPBanned(ip)
}

func (i *ipService) BanIP(ip, reason string, expireTime time.Time) error {
	logrus.Infof("IP %v 已被封禁 %v，原因 %v", ip, expireTime.String(), reason)
	return i.repo.BanIP(ip, reason, expireTime)
}

func (i *ipService) UnbanIP(ip string) error {
	logrus.Infof("IP %v 已被解封", ip)
	return i.repo.UnbanIP(ip)
}
