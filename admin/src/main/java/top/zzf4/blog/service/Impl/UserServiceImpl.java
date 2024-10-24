package top.zzf4.blog.service.Impl;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.extension.service.impl.ServiceImpl;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import top.zzf4.blog.entity.model.User;
import top.zzf4.blog.mapper.UserMapper;
import top.zzf4.blog.service.UserService;
import top.zzf4.blog.utils.JwtUtils;

@Service
public class UserServiceImpl extends ServiceImpl<UserMapper, User> implements UserService {
    @Autowired
    private UserMapper userMapper;

    @Override
    public String login(String username, String password) {
        // 通过用户名查询密码
        User user = this.getOne(new LambdaQueryWrapper<User>().eq(User::getUsername, username));
        if (user == null) throw new RuntimeException("用户不存在");

        // 验证密码
        if (!JwtUtils.verifyByBCrypt(password, user.getPassword())) {
            throw new RuntimeException("密码错误");
        }

        return JwtUtils.createToken(user);
    }

}
