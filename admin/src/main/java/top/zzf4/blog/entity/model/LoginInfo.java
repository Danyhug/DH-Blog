package top.zzf4.blog.entity.model;

import lombok.Data;

import java.time.LocalDateTime;

@Data
public class LoginInfo {

    private Long id;

    private User user;

    private LocalDateTime loginTime;

    private String ip;
    private String city;

    // 省略构造器、getter和setter方法
}