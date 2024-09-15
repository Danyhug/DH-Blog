package top.zzf4.blog.config;

import lombok.Data;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.stereotype.Component;

// 七牛云配置
@Data
@Component
@ConfigurationProperties(prefix = "qiniu")
public class QiniuProperties {
    private String accessKey;
    private String secretKey;
    private String bucket;
    // 默认展示的图片文件夹名
    private String defaultImageName;
}
