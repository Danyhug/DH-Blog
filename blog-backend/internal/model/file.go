package model

// File 代表一个用户的文件或文件夹实体
type File struct {
	BaseModel // 使用自定义的BaseModel替代gorm.Model
	// --- 基础信息 ---

	// UserID 文件所属用户的ID
	// 用于权限隔离，确保用户只能看到自己的文件
	// 当前简化实现中：作为参数传递，但未实际用于权限控制
	UserID uint64 `gorm:"index" json:"user_id,omitempty"`

	// ParentID 父文件夹的ID或路径
	// 如果为空或根目录标识，表示它位于用户的根目录
	// 当前简化实现中：使用字符串存储相对路径
	ParentID string `json:"parent_id,omitempty"`

	// Name 文件或文件夹的名称
	// 例如: "我的文档.docx", "学习资料"
	// 当前简化实现中：实际使用，显示文件名
	Name string `gorm:"type:varchar(255)" json:"name"`

	// IsFolder 标记这是一个文件还是文件夹
	// 布尔值，true 表示是文件夹，false 表示是文件
	// 当前简化实现中：实际使用，区分文件和文件夹
	IsFolder bool `gorm:"not null" json:"is_folder"`

	// --- 文件专属信息 ---

	// Size 文件的大小，单位是字节 (bytes)
	// 文件夹的大小可以设为 0，或者在需要时动态计算其下所有文件的总和
	// 当前简化实现中：实际使用，存储文件大小
	Size int64 `gorm:"not null" json:"size"`

	// MimeType 文件的媒体类型，例如 "image/jpeg", "application/pdf"
	// 这个字段对于前端展示不同的文件图标非常有用
	// 当前简化实现中：未使用，留作UI展示用
	MimeType string `gorm:"type:varchar(100)" json:"mime_type,omitempty"`

	// --- 为未来扩展预留的字段 ---

	// FileHash 文件的内容哈希值 (例如 SHA256)
	// 这是实现"秒传"功能的核心。即使现在不用，预留好位置总没错。
	// 对于文件夹，此字段为空。
	// 当前简化实现中：未使用，留作秒传功能用
	FileHash string `gorm:"type:varchar(255);index" json:"-"`

	// StoragePath 文件在后端存储系统（如本地磁盘、对象存储）中的实际存储路径或唯一标识
	// 对于文件夹，此字段为空。
	// 当前简化实现中：仅在GetDownloadInfo中临时使用，返回完整文件路径
	StoragePath string `gorm:"type:varchar(1024)" json:"-"`
}

// TableName 自定义 GORM 在数据库中映射的表名
// 当前简化实现中：未使用，留作将来数据库集成用
func (File) TableName() string {
	return "files"
}
