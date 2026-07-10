package share

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"dh-blog/internal/model"
	filesmodule "dh-blog/internal/modules/files"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type CreateShareRequest struct {
	FileKey          string     `json:"file_key" binding:"required"`
	Password         string     `json:"password,omitempty"`
	ExpireAt         *time.Time `json:"expire_at,omitempty"`
	MaxDownloadCount *int       `json:"max_download_count,omitempty"`
}

type ShareInfoResponse struct {
	ShareID       string     `json:"share_id"`
	FileName      string     `json:"file_name"`
	FileSize      int64      `json:"file_size"`
	HasPassword   bool       `json:"has_password"`
	ExpireAt      *time.Time `json:"expire_at,omitempty"`
	IsExpired     bool       `json:"is_expired"`
	ViewCount     int64      `json:"view_count"`
	DownloadCount int64      `json:"download_count"`
	CreatedAt     string     `json:"create_time"`
}

type VerifyPasswordResponse struct {
	Valid         bool   `json:"valid"`
	DownloadToken string `json:"download_token,omitempty"`
	ExpiresIn     int    `json:"expires_in,omitempty"`
}

type downloadToken struct {
	ShareID         string
	CreatedAt       time.Time
	ExpiresAt       time.Time
	DownloadCounted bool
	mu              sync.Mutex
}

type tokenManager struct {
	store    sync.Map
	expiry   time.Duration
	ticker   *time.Ticker
	stop     chan struct{}
	done     chan struct{}
	stopOnce sync.Once
}

func newTokenManager(expiry, cleanupInterval time.Duration) *tokenManager {
	m := &tokenManager{
		expiry: expiry,
		ticker: time.NewTicker(cleanupInterval),
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
	go m.cleanupLoop()
	return m
}

func (m *tokenManager) cleanupLoop() {
	defer close(m.done)
	for {
		select {
		case now := <-m.ticker.C:
			m.store.Range(func(key, value any) bool {
				token, ok := value.(*downloadToken)
				if ok && now.After(token.ExpiresAt) {
					m.store.Delete(key)
				}
				return true
			})
		case <-m.stop:
			return
		}
	}
}

func (m *tokenManager) shutdown() {
	m.stopOnce.Do(func() {
		m.ticker.Stop()
		close(m.stop)
		<-m.done
	})
}

// Service is the share module's public business contract.
type Service interface {
	CreateShare(ctx context.Context, req *CreateShareRequest) (*Share, error)
	GetShareInfo(ctx context.Context, shareID string) (*ShareInfoResponse, error)
	GetShareDetail(ctx context.Context, id int) (*Share, error)
	VerifyPassword(ctx context.Context, shareID, password string) (*VerifyPasswordResponse, error)
	DownloadWithToken(ctx context.Context, shareID, token string, clientIP, userAgent, referer string, preview bool) (*filesmodule.File, error)
	Download(ctx context.Context, shareID string, clientIP, userAgent, referer string) (*filesmodule.File, error)
	ListShares(ctx context.Context, page, pageSize int) ([]*Share, int64, error)
	DeleteShare(ctx context.Context, id int) error
	GetShareAccessLogs(ctx context.Context, shareID string, page, pageSize int) ([]*ShareAccessLog, int64, error)
	RecordAccess(ctx context.Context, shareID, actionType, clientIP, userAgent, referer string) error
}

type shareService struct {
	shareRepo     *Repository
	accessLogRepo *AccessLogRepository
	fileService   filesmodule.Service
	tokens        *tokenManager
}

func newService(
	shareRepo *Repository,
	accessLogRepo *AccessLogRepository,
	fileService filesmodule.Service,
) *shareService {
	return &shareService{
		shareRepo:     shareRepo,
		accessLogRepo: accessLogRepo,
		fileService:   fileService,
		tokens:        newTokenManager(5*time.Minute, time.Minute),
	}
}

func (s *shareService) shutdown() {
	s.tokens.shutdown()
}

func generateShareID() (string, error) {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func generateDownloadToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func hashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

func checkPassword(hashedPassword, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil
}

func (s *shareService) CreateShare(ctx context.Context, req *CreateShareRequest) (*Share, error) {
	file, err := s.fileService.GetDownloadInfo(ctx, 1, req.FileKey)
	if err != nil {
		logrus.Errorf("获取文件信息失败: %v", err)
		return nil, errors.New("文件不存在")
	}
	if file.IsFolder {
		return nil, errors.New("暂不支持分享文件夹")
	}

	shareID, err := generateShareID()
	if err != nil {
		logrus.Errorf("生成分享ID失败: %v", err)
		return nil, errors.New("创建分享失败")
	}

	var hashedPassword string
	if req.Password != "" {
		hashedPassword, err = hashPassword(req.Password)
		if err != nil {
			logrus.Errorf("密码加密失败: %v", err)
			return nil, errors.New("创建分享失败")
		}
	}

	share := &Share{
		ShareID:          shareID,
		FileKey:          req.FileKey,
		Password:         hashedPassword,
		ExpireAt:         req.ExpireAt,
		MaxDownloadCount: req.MaxDownloadCount,
		ViewCount:        0,
		DownloadCount:    0,
	}
	if err := s.shareRepo.Create(ctx, share); err != nil {
		logrus.Errorf("创建分享记录失败: %v", err)
		return nil, errors.New("创建分享失败")
	}
	return share, nil
}

func (s *shareService) GetShareInfo(ctx context.Context, shareID string) (*ShareInfoResponse, error) {
	share, err := s.shareRepo.FindByShareID(ctx, shareID)
	if err != nil {
		return nil, errors.New("分享不存在")
	}

	file, err := s.fileService.GetDownloadInfo(ctx, 1, share.FileKey)
	if err != nil {
		logrus.Errorf("获取文件信息失败: %v", err)
		return nil, errors.New("文件不存在")
	}
	if err := s.shareRepo.IncrementViewCount(ctx, shareID); err != nil {
		logrus.Warnf("增加查看次数失败: %v", err)
	}

	return &ShareInfoResponse{
		ShareID:       share.ShareID,
		FileName:      file.Name,
		FileSize:      file.Size,
		HasPassword:   share.HasPassword(),
		ExpireAt:      share.ExpireAt,
		IsExpired:     share.IsExpired(),
		ViewCount:     share.ViewCount + 1,
		DownloadCount: share.DownloadCount,
		CreatedAt:     share.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *shareService) GetShareDetail(ctx context.Context, id int) (*Share, error) {
	return s.shareRepo.FindByID(ctx, id)
}

func (s *shareService) VerifyPassword(ctx context.Context, shareID, password string) (*VerifyPasswordResponse, error) {
	share, err := s.shareRepo.FindByShareID(ctx, shareID)
	if err != nil {
		return nil, errors.New("分享不存在")
	}
	if err := s.validateShare(share); err != nil {
		return nil, err
	}

	if share.HasPassword() && !checkPassword(share.Password, password) {
		return &VerifyPasswordResponse{Valid: false}, nil
	}

	token, err := s.createDownloadToken(shareID)
	if err != nil {
		return nil, errors.New("生成下载令牌失败")
	}
	return &VerifyPasswordResponse{
		Valid:         true,
		DownloadToken: token,
		ExpiresIn:     int(s.tokens.expiry.Seconds()),
	}, nil
}

func (s *shareService) createDownloadToken(shareID string) (string, error) {
	token, err := generateDownloadToken()
	if err != nil {
		return "", err
	}
	now := time.Now()
	s.tokens.store.Store(token, &downloadToken{
		ShareID:   shareID,
		CreatedAt: now,
		ExpiresAt: now.Add(s.tokens.expiry),
	})
	return token, nil
}

func (s *shareService) validateDownloadToken(shareID, token string) bool {
	value, ok := s.tokens.store.Load(token)
	if !ok {
		return false
	}
	dt, ok := value.(*downloadToken)
	if !ok {
		return false
	}
	if time.Now().After(dt.ExpiresAt) {
		s.tokens.store.Delete(token)
		return false
	}
	return dt.ShareID == shareID
}

func (s *shareService) DownloadWithToken(ctx context.Context, shareID, token string, clientIP, userAgent, referer string, preview bool) (*filesmodule.File, error) {
	if !s.validateDownloadToken(shareID, token) {
		return nil, errors.New("下载令牌无效或已过期")
	}

	share, err := s.shareRepo.FindByShareID(ctx, shareID)
	if err != nil {
		return nil, errors.New("分享不存在")
	}
	if err := s.validateShare(share); err != nil {
		return nil, err
	}

	file, err := s.fileService.GetDownloadInfo(ctx, 1, share.FileKey)
	if err != nil {
		logrus.Errorf("获取文件信息失败: %v", err)
		return nil, errors.New("文件不存在")
	}

	if !preview {
		if value, ok := s.tokens.store.Load(token); ok {
			if dt, ok := value.(*downloadToken); ok {
				dt.mu.Lock()
				if !dt.DownloadCounted {
					if err := s.shareRepo.IncrementDownloadCount(ctx, shareID); err != nil {
						logrus.Warnf("增加下载次数失败: %v", err)
					}
					dt.DownloadCounted = true
					go func() {
						if err := s.RecordAccess(context.Background(), shareID, ShareActionDownload, clientIP, userAgent, referer); err != nil {
							logrus.Warnf("记录下载日志失败: %v", err)
						}
					}()
				}
				dt.mu.Unlock()
			}
		}
	}
	return file, nil
}

func (s *shareService) Download(ctx context.Context, shareID string, clientIP, userAgent, referer string) (*filesmodule.File, error) {
	share, err := s.shareRepo.FindByShareID(ctx, shareID)
	if err != nil {
		return nil, errors.New("分享不存在")
	}
	if share.HasPassword() {
		return nil, errors.New("此分享需要密码验证")
	}
	if err := s.validateShare(share); err != nil {
		return nil, err
	}

	file, err := s.fileService.GetDownloadInfo(ctx, 1, share.FileKey)
	if err != nil {
		logrus.Errorf("获取文件信息失败: %v", err)
		return nil, errors.New("文件不存在")
	}
	if err := s.shareRepo.IncrementDownloadCount(ctx, shareID); err != nil {
		logrus.Warnf("增加下载次数失败: %v", err)
	}
	go func() {
		if err := s.RecordAccess(context.Background(), shareID, ShareActionDownload, clientIP, userAgent, referer); err != nil {
			logrus.Warnf("记录下载日志失败: %v", err)
		}
	}()
	return file, nil
}

func (s *shareService) validateShare(share *Share) error {
	if share.IsExpired() {
		return errors.New("分享已过期")
	}
	if share.IsDownloadLimitReached() {
		return errors.New("下载次数已达上限")
	}
	return nil
}

func (s *shareService) ListShares(ctx context.Context, page, pageSize int) ([]*Share, int64, error) {
	return s.shareRepo.ListByPage(ctx, page, pageSize)
}

func (s *shareService) DeleteShare(ctx context.Context, id int) error {
	return s.shareRepo.Delete(ctx, id)
}

func (s *shareService) GetShareAccessLogs(ctx context.Context, shareID string, page, pageSize int) ([]*ShareAccessLog, int64, error) {
	return s.accessLogRepo.ListByShareID(ctx, shareID, page, pageSize)
}

func (s *shareService) RecordAccess(ctx context.Context, shareID, actionType, clientIP, userAgent, referer string) error {
	log := &ShareAccessLog{
		ShareID:    shareID,
		ActionType: actionType,
		IP:         clientIP,
		UserAgent:  userAgent,
		Referer:    referer,
		CreatedAt:  model.JSONTime{Time: time.Now()},
	}
	if err := s.accessLogRepo.Create(ctx, log); err != nil {
		return fmt.Errorf("记录访问日志失败: %w", err)
	}
	return nil
}
