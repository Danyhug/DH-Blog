package top.zzf4.blog;

import lombok.extern.log4j.Log4j2;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import top.zzf4.blog.entity.model.User;
import top.zzf4.blog.mapper.UserMapper;
import top.zzf4.blog.utils.JwtUtils;

@Log4j2
@SpringBootTest
public class JwtTest {

    @Autowired
    private UserMapper userMapper;

    @Test
    public void test() {
        User user = User.builder().username("zzf4").id(1L).build();
        String token = JwtUtils.createToken(user);
        log.info("测试JWT生成token {} - {}", user, token);

        log.info("测试JWT解析token {}", JwtUtils.parseToken(token));
    }

    // @Test
    // public void addUser() {
    //     String username = "admin";
    //     String rawPassword = "admin";
    //     String hashPassword = JwtUtils.encodeByBCrypt(rawPassword);
    //     userMapper.insert(User.builder().username(username).password(hashPassword).build());
    // }
}
