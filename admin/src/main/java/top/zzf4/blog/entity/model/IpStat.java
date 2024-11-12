package top.zzf4.blog.entity.model;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Builder;
import lombok.Data;

@Data
@Builder
@TableName("ip_stats")
public class IpStat {
    @TableId(value = "id", type = IdType.AUTO)
    private Integer id;

    @TableField("ip_address")
    private String ipAddress;

    @TableField(value = "access_count")
    private Integer accessCount;

    @TableField(value = "banned_count")
    private Integer bannedCount;
}