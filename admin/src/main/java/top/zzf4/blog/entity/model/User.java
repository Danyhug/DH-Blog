package top.zzf4.blog.entity.model;

import lombok.Data;

import java.time.LocalDateTime;

@Data
public class User {
    private Long id;

    private String username;

    private String password;

    private LocalDateTime registerDate;

    // 省略构造器、getter和setter方法
}