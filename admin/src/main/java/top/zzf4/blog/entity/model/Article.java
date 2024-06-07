package top.zzf4.blog.entity.model;

import com.fasterxml.jackson.annotation.JsonFormat;
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

    @JsonFormat(pattern = "yyyy-MM-dd HH:mm:ss")
    private LocalDateTime publishDate;

    @JsonFormat(pattern = "yyyy-MM-dd HH:mm:ss")
    private LocalDateTime updateDate;

    private int views;

    private byte wordNum;

    private List<Tag> tags;

    private String thumbnail_url; // 缩略图url
}