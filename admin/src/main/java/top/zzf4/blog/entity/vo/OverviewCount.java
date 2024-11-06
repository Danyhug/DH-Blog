package top.zzf4.blog.entity.vo;

import lombok.Builder;
import lombok.Data;

// 总览
@Builder
@Data
public class OverviewCount {
    private Long articleCount;
    private Long categoryCount;
    private Long tagCount;
    private Long commentCount;
}
