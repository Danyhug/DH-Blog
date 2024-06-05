package top.zzf4.blog.entity.model;

import lombok.Builder;
import lombok.Data;

import java.time.LocalDateTime;
import java.util.List;

@Data
public class Article {

    private Long id;

    private String title;

    private String content;

    private Integer categoryId;

    private LocalDateTime publishDate;

    private LocalDateTime updateDate;

    private int views;

    private byte wordNum; // TINYINT映射为byte

    private List<Tag> tags;
}