export interface SystemConfig {
    id?: number;
    blog_title?: string;
    signature?: string;
    avatar?: string;
    github_link?: string;
    bilibili_link?: string;
    open_blog?: boolean;
    open_comment?: boolean;
    comment_email_notify?: boolean;
    smtp_host?: string;
    smtp_port?: number;
    smtp_user?: string;
    smtp_pass?: string;
    smtp_sender?: string;
    ai_api_url?: string;
    ai_api_key?: string;
    ai_model?: string;
    ai_prompt?: string;
}
