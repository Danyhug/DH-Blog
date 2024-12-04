package top.zzf4.blog.entity.model;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@Builder
@TableName("ip_stats")
@NoArgsConstructor
@AllArgsConstructor
public class IpStat {
    @TableId(value = "id", type = IdType.AUTO)
    private Integer id;

    @TableField("ip_address")
    private String ipAddress;

    private String city;

    @TableField(value = "access_count")
    private Integer accessCount;

    @TableField(value = "banned_count")
    private Integer bannedCount;

    @TableField(value = "ban_status")
    private Integer banStatus;
}