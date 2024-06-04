package top.zzf4.blog.entity.dto;

import lombok.Data;

@Data
public class ArticlePageDTO {
    private Integer pageNum;
    private Integer pageSize;
    private Integer categoryId;
}
