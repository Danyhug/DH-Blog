package top.zzf4.blog.service;

import com.baomidou.mybatisplus.extension.service.IService;
import top.zzf4.blog.entity.model.User;

public interface UserService extends IService<User> {
    // 登录成功，返回 token
    String login(String username, String password);
}
