package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"dh-blog/internal/model"
	"dh-blog/internal/repository"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// CreateShareRequest 创建分享请求
type CreateShareRequest struct {
	FileKey          string     `json:"file_key" binding:"required"`  // 文件ID
	Password         string     `json:"password,omitempty"`           // 访问密码（可选）
	ExpireAt         *time.Time `json:"expire_at,omitempty"`          // 过期时间（可选）
	MaxDownloadCount *int       `json:"max_download_count,omitempty"` // 最大下载次数（可选）
}

// ShareInfoResponse 分享信息响应
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

// VerifyPasswordResponse 密码验证响应
type VerifyPasswordResponse struct {
	Valid         bool   `json:"valid"`
	DownloadToken string `json:"download_token,omitempty"` // 临时下载令牌
	ExpiresIn     int    `json:"expires_in,omitempty"`     // 令牌有效期（秒）
}

// downloadToken 下载令牌信息
type downloadToken struct {
	ShareID   string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// tokenStore 令牌存储
var (
	tokenStore     sync.Map
	tokenExpiry    = 5 * time.Minute // 令牌有效期5分钟
	cleanupTicker  *time.Ticker
	cleanupStarted bool
	cleanupMu      sync.Mutex
)

// IShareService 分享服务接口
type IShareService interface {
	// CreateShare 创建分享链接
	CreateShare(ctx context.Context, req *CreateShareRequest) (*model.Share, error)
	// GetShareInfo 获取分享信息（用于公开访问页）
	GetShareInfo(ctx context.Context, shareID string) (*ShareInfoResponse, error)
	// GetShareDetail 获取分享详情（用于管理）
	GetShareDetail(ctx context.Context, id int) (*model.Share, error)
	// VerifyPassword 验证分享密码并返回下载令牌
	VerifyPassword(ctx context.Context, shareID, password string) (*VerifyPasswordResponse, error)
	// DownloadWithToken 使用令牌下载分享文件
	// preview 为 true 时表示预览模式（音视频流式传输），不消耗令牌
	DownloadWithToken(ctx context.Context, shareID, token string, clientIP, userAgent, referer string, preview bool) (*model.File, error)
	// Download 下载分享文件（无密码分享使用）
	Download(ctx context.Context, shareID string, clientIP, userAgent, referer string) (*model.File, error)
	// ListShares 分页获取分享列表
	ListShares(ctx context.Context, page, pageSize int) ([]*model.Share, int64, error)
	// DeleteShare 删除分享
	DeleteShare(ctx context.Context, id int) error
	// GetShareAccessLogs 获取分享访问日志
	GetShareAccessLogs(ctx context.Context, shareID string, page, pageSize int) ([]*model.ShareAccessLog, int64, error)
	// RecordAccess 记录访问日志
	RecordAccess(ctx context.Context, shareID, actionType, clientIP, userAgent, referer string) error
}

// shareService 分享服务实现
type shareService struct {
	shareRepo     repository.IShareRepository
	accessLogRepo repository.IShareAccessLogRepository
	fileService   IFileService
}

// NewShareService 创建分享服务
func NewShareService(
	shareRepo repository.IShareRepository,
	accessLogRepo repository.IShareAccessLogRepository,
	fileService IFileService,
) IShareService {
	// 启动令牌清理协程
	startTokenCleanup()

	return &shareService{
		shareRepo:     shareRepo,
		accessLogRepo: accessLogRepo,
		fileService:   fileService,
	}
}

// startTokenCleanup 启动令牌清理协程
func startTokenCleanup() {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()

	if cleanupStarted {
		return
	}

	cleanupStarted = true
	cleanupTicker = time.NewTicker(1 * time.Minute)

	go func() {
		for range cleanupTicker.C {
			now := time.Now()
			tokenStore.Range(func(key, value interface{}) bool {
				if token, ok := value.(*downloadToken); ok {
					if now.After(token.ExpiresAt) {
						tokenStore.Delete(key)
					}
				}
				return true
			})
		}
	}()
}

// generateShareID 生成分享ID（8位随机字符串）
func generateShareID() (string, error) {
	bytes := make([]byte, 4) // 4 bytes = 8 hex chars
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// generateDownloadToken 生成下载令牌（32位随机字符串）
func generateDownloadToken() (string, error) {
	bytes := make([]byte, 16) // 16 bytes = 32 hex chars
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// hashPassword 对密码进行哈希
func hashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// checkPassword 验证密码
func checkPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// CreateShare 创建分享链接
func (s *shareService) CreateShare(ctx context.Context, req *CreateShareRequest) (*model.Share, error) {
	// 验证文件是否存在
	file, err := s.fileService.GetDownloadInfo(ctx, 1, req.FileKey) // userID暂时用1
	if err != nil {
		logrus.Errorf("获取文件信息失败: %v", err)
		return nil, errors.New("文件不存在")
	}

	// 检查是否是文件夹（暂不支持文件夹分享）
	if file.IsFolder {
		return nil, errors.New("暂不支持分享文件夹")
	}

	// 生成分享ID
	shareID, err := generateShareID()
	if err != nil {
		logrus.Errorf("生成分享ID失败: %v", err)
		return nil, errors.New("创建分享失败")
	}

	// 处理密码
	var hashedPassword string
	if req.Password != "" {
		hashedPassword, err = hashPassword(req.Password)
		if err != nil {
			logrus.Errorf("密码加密失败: %v", err)
			return nil, errors.New("创建分享失败")
		}
	}

	// 创建分享记录
	share := &model.Share{
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

// GetShareInfo 获取分享信息（用于公开访问页）
func (s *shareService) GetShareInfo(ctx context.Context, shareID string) (*ShareInfoResponse, error) {
	share, err := s.shareRepo.FindByShareID(ctx, shareID)
	if err != nil {
		return nil, errors.New("分享不存在")
	}

	// 获取文件信息
	file, err := s.fileService.GetDownloadInfo(ctx, 1, share.FileKey)
	if err != nil {
		logrus.Errorf("获取文件信息失败: %v", err)
		return nil, errors.New("文件不存在")
	}

	// 增加查看次数
	if err := s.shareRepo.IncrementViewCount(ctx, shareID); err != nil {
		logrus.Warnf("增加查看次数失败: %v", err)
	}

	response := &ShareInfoResponse{
		ShareID:       share.ShareID,
		FileName:      file.Name,
		FileSize:      file.Size,
		HasPassword:   share.HasPassword(),
		ExpireAt:      share.ExpireAt,
		IsExpired:     share.IsExpired(),
		ViewCount:     share.ViewCount + 1, // 返回更新后的计数
		DownloadCount: share.DownloadCount,
		CreatedAt:     share.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	return response, nil
}

// GetShareDetail 获取分享详情（用于管理）
func (s *shareService) GetShareDetail(ctx context.Context, id int) (*model.Share, error) {
	return s.shareRepo.FindByID(ctx, id)
}

// VerifyPassword 验证分享密码并返回下载令牌
func (s *shareService) VerifyPassword(ctx context.Context, shareID, password string) (*VerifyPasswordResponse, error) {
	share, err := s.shareRepo.FindByShareID(ctx, shareID)
	if err != nil {
		return nil, errors.New("分享不存在")
	}

	// 检查分享有效性
	if err := s.validateShare(share); err != nil {
		return nil, err
	}

	// 如果没有设置密码，直接生成令牌
	if !share.HasPassword() {
		token, err := s.createDownloadToken(shareID)
		if err != nil {
			return nil, errors.New("生成下载令牌失败")
		}
		return &VerifyPasswordResponse{
			Valid:         true,
			DownloadToken: token,
			ExpiresIn:     int(tokenExpiry.Seconds()),
		}, nil
	}

	// 验证密码
	if !checkPassword(share.Password, password) {
		return &VerifyPasswordResponse{Valid: false}, nil
	}

	// 密码正确，生成下载令牌
	token, err := s.createDownloadToken(shareID)
	if err != nil {
		return nil, errors.New("生成下载令牌失败")
	}

	return &VerifyPasswordResponse{
		Valid:         true,
		DownloadToken: token,
		ExpiresIn:     int(tokenExpiry.Seconds()),
	}, nil
}

// createDownloadToken 创建下载令牌
func (s *shareService) createDownloadToken(shareID string) (string, error) {
	token, err := generateDownloadToken()
	if err != nil {
		return "", err
	}

	now := time.Now()
	tokenStore.Store(token, &downloadToken{
		ShareID:   shareID,
		CreatedAt: now,
		ExpiresAt: now.Add(tokenExpiry),
	})

	return token, nil
}

// validateDownloadToken 验证下载令牌
func (s *shareService) validateDownloadToken(shareID, token string) bool {
	value, ok := tokenStore.Load(token)
	if !ok {
		return false
	}

	dt, ok := value.(*downloadToken)
	if !ok {
		return false
	}

	// 检查令牌是否过期
	if time.Now().After(dt.ExpiresAt) {
		tokenStore.Delete(token)
		return false
	}

	// 检查令牌是否属于该分享
	if dt.ShareID != shareID {
		return false
	}

	return true
}

// DownloadWithToken 使用令牌下载分享文件
func (s *shareService) DownloadWithToken(ctx context.Context, shareID, token string, clientIP, userAgent, referer string, preview bool) (*model.File, error) {
	// 验证令牌
	if !s.validateDownloadToken(shareID, token) {
		return nil, errors.New("下载令牌无效或已过期")
	}

	// 获取分享信息
	share, err := s.shareRepo.FindByShareID(ctx, shareID)
	if err != nil {
		return nil, errors.New("分享不存在")
	}

	// 检查分享有效性
	if err := s.validateShare(share); err != nil {
		return nil, err
	}

	// 获取文件信息
	file, err := s.fileService.GetDownloadInfo(ctx, 1, share.FileKey)
	if err != nil {
		logrus.Errorf("获取文件信息失败: %v", err)
		return nil, errors.New("文件不存在")
	}

	// 预览模式不增加下载次数，不删除令牌（支持音视频流式多次请求）
	if !preview {
		// 增加下载次数
		if err := s.shareRepo.IncrementDownloadCount(ctx, shareID); err != nil {
			logrus.Warnf("增加下载次数失败: %v", err)
		}

		// 记录下载日志
		go func() {
			if err := s.RecordAccess(context.Background(), shareID, model.ShareActionDownload, clientIP, userAgent, referer); err != nil {
				logrus.Warnf("记录下载日志失败: %v", err)
			}
		}()

		// 下载成功后删除令牌（一次性使用）
		tokenStore.Delete(token)
	}

	return file, nil
}

// Download 下载分享文件（无密码分享使用）
func (s *shareService) Download(ctx context.Context, shareID string, clientIP, userAgent, referer string) (*model.File, error) {
	share, err := s.shareRepo.FindByShareID(ctx, shareID)
	if err != nil {
		return nil, errors.New("分享不存在")
	}

	// 如果有密码，不允许直接下载
	if share.HasPassword() {
		return nil, errors.New("此分享需要密码验证")
	}

	// 检查分享有效性
	if err := s.validateShare(share); err != nil {
		return nil, err
	}

	// 获取文件信息
	file, err := s.fileService.GetDownloadInfo(ctx, 1, share.FileKey)
	if err != nil {
		logrus.Errorf("获取文件信息失败: %v", err)
		return nil, errors.New("文件不存在")
	}

	// 增加下载次数
	if err := s.shareRepo.IncrementDownloadCount(ctx, shareID); err != nil {
		logrus.Warnf("增加下载次数失败: %v", err)
	}

	// 记录下载日志
	go func() {
		if err := s.RecordAccess(context.Background(), shareID, model.ShareActionDownload, clientIP, userAgent, referer); err != nil {
			logrus.Warnf("记录下载日志失败: %v", err)
		}
	}()

	return file, nil
}

// validateShare 验证分享有效性
func (s *shareService) validateShare(share *model.Share) error {
	// 检查是否过期
	if share.IsExpired() {
		return errors.New("分享已过期")
	}

	// 检查下载次数限制
	if share.IsDownloadLimitReached() {
		return errors.New("下载次数已达上限")
	}

	return nil
}

// ListShares 分页获取分享列表
func (s *shareService) ListShares(ctx context.Context, page, pageSize int) ([]*model.Share, int64, error) {
	return s.shareRepo.ListByPage(ctx, page, pageSize)
}

// DeleteShare 删除分享
func (s *shareService) DeleteShare(ctx context.Context, id int) error {
	return s.shareRepo.Delete(ctx, id)
}

// GetShareAccessLogs 获取分享访问日志
func (s *shareService) GetShareAccessLogs(ctx context.Context, shareID string, page, pageSize int) ([]*model.ShareAccessLog, int64, error) {
	return s.accessLogRepo.ListByShareID(ctx, shareID, page, pageSize)
}

// RecordAccess 记录访问日志
func (s *shareService) RecordAccess(ctx context.Context, shareID, actionType, clientIP, userAgent, referer string) error {
	log := &model.ShareAccessLog{
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
