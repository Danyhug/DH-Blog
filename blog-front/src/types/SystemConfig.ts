// 整体系统配置
export interface SystemConfig {
    id?: number;
    // 博客基本配置
    blog_title?: string;
    signature?: string;
    avatar?: string;
    github_link?: string;
    bilibili_link?: string;
    open_blog?: boolean;
    open_comment?: boolean;
    
    // 邮件配置
    comment_email_notify?: boolean;
    smtp_host?: string;
    smtp_port?: number;
    smtp_user?: string;
    smtp_pass?: string;
    smtp_sender?: string;
    
    // AI配置
    ai_api_url?: string;
    ai_api_key?: string;
    ai_model?: string;
    ai_prompt?: string;
    
    // 存储配置
    file_storage_path?: string; // 文件存储路径
    webdav_chunk_size?: number; // WebDAV分片大小(KB)
}

// 博客基本配置
export interface BlogConfig {
    blog_title?: string;
    signature?: string;
    avatar?: string;
    github_link?: string;
    bilibili_link?: string;
    open_blog?: boolean;
    open_comment?: boolean;
}

// 邮件配置
export interface EmailConfig {
    comment_email_notify?: boolean;
    smtp_host?: string;
    smtp_port?: number;
    smtp_user?: string;
    smtp_pass?: string;
    smtp_sender?: string;
}

// AI配置
export interface AIConfig {
    ai_api_url?: string;
    ai_api_key?: string;
    ai_model?: string;
}

// 存储配置
export interface StorageConfig {
    file_storage_path?: string;
    webdav_chunk_size?: number; // WebDAV分片大小(KB)
}
