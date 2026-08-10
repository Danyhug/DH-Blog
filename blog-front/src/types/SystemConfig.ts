// 博客基本配置
export interface BlogConfig {
    blog_title?: string;
    signature?: string;
    avatar?: string;
    github_link?: string;
    bilibili_link?: string;
    open_comment?: boolean;
}

// 站点公开配置（前台展示用，无需鉴权）
export interface SiteConfig {
    blog_title: string;
    signature: string;
    avatar: string;
    github_link: string;
    bilibili_link: string;
    open_comment: boolean;
}

// AI配置
export interface AIConfig {
    ai_api_url?: string;
    ai_api_key?: string;
    ai_model?: string;
}

// AI提示词，key 对应后端 system_settings 中的键
export interface AIPrompt {
    key: string;
    label: string;
    prompt: string;
}

// 存储配置
export interface StorageConfig {
    file_storage_path?: string;
    webdav_chunk_size?: number; // WebDAV分片大小(KB)
}
