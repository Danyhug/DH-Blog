package top.zzf4.blog.entity.model;

import lombok.Data;

import java.time.LocalDateTime;

@Data
public class User {
    private Long id;

    private String username;

    private String password;

    private LocalDateTime createTime;
}