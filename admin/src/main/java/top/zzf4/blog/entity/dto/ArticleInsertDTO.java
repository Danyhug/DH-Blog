package top.zzf4.blog.entity.dto;

import lombok.Data;

@Data
public class ArticleInsertDTO {
    private String title;
    private String content;
    private String[] tags;
    private Integer wordNum;
    private Integer categoryId;
}
