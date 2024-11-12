package top.zzf4.blog.entity.model;

import cn.hutool.core.date.DateTime;
import com.baomidou.mybatisplus.annotation.*;
import lombok.Data;

import java.time.LocalDateTime;
import java.util.Date;

@Data
@TableName("access_logs")
public class AccessLog {
    @TableId(value = "id", type = IdType.AUTO)
    private Long id;

    @TableField("ip_address")
    private String ipAddress;

    @TableField(value = "access_time", fill = FieldFill.INSERT)
    private LocalDateTime accessTime;

    @TableField("user_agent")
    private String userAgent;

    @TableField("request_url")
    private String requestUrl;
}