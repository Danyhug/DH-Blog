package top.zzf4.blog.entity.model;

import lombok.Data;

import java.time.LocalDateTime;

@Data
public class Article {

    private Long id;

    private String title;

    private String content;

    private Category category;

    private LocalDateTime publishDate;

    private LocalDateTime updateDate;

    private int views;

    private byte wordNum; // TINYINT映射为byte

    // 省略构造器、getter和setter方法
}