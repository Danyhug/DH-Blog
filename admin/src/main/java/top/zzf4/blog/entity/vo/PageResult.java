package top.zzf4.blog.entity.vo;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.io.Serializable;
import java.util.List;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class PageResult<T> implements Serializable {
    // 总页数
    private Long total;
    // 当前页数
    private Long curr;
    // 数据列表
    private List<T> list;
}
