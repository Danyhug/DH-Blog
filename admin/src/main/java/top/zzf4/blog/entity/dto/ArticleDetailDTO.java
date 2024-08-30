package top.zzf4.blog.entity.dto;

import lombok.Data;
import top.zzf4.blog.entity.model.Category;

import java.time.LocalDateTime;

@Data
public class ArticleDetailDTO {
    private Long id;

    private String title;

    private String content;

    private Category category;

    private LocalDateTime publishDate;

    private LocalDateTime updateTimee;

    private int views;

    private byte wordNum; // TINYINT映射为byte
}
