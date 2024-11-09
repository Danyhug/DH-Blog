package top.zzf4.blog.entity.model;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Data;
import java.util.Date;

@Data
@TableName("daily_stats")
public class DailyStats {

    @TableId(value = "id", type = IdType.AUTO)
    private Integer id; // 唯一标识每一天的统计数据

    private Date date; // 记录日期

    private Integer visitCount; // 记录当天的访问量

    private Integer articleCount; // 记录当天的文章发布数量

    private Integer commentCount; // 记录当天的评论数量

    private Integer tagCount; // 记录当天的标签发布数量

    @Override
    public String toString() {
        return "DailyStats{" +
                "id=" + id +
                ", date=" + date +
                ", visitCount=" + visitCount +
                ", articleCount=" + articleCount +
                ", commentCount=" + commentCount +
                ", tagCount=" + tagCount +
                '}';
    }
}
