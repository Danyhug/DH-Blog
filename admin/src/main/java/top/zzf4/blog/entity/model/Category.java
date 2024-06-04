package top.zzf4.blog.entity.model;

import lombok.Data;

import java.time.LocalDateTime;

@Data
public class Category {

    private Long id;

    private String name;

    private String slug;

    private LocalDateTime createdAt;
    private LocalDateTime updatedAt;

    // 省略构造器、getter和setter方法
}