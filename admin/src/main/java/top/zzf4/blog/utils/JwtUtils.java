package top.zzf4.blog.utils;

import cn.hutool.jwt.JWTPayload;
import cn.hutool.jwt.JWTUtil;
import top.zzf4.blog.entity.model.User;

import java.util.HashMap;

public class JwtUtils {
    private static final String SECRET = "zzf4";

    /**
     * 生成token
     * @param user
     * @return
     */
    public static String createToken(User user) {
        return JWTUtil.createToken(
                new HashMap<>() {{
                    put("username", user.getUsername());
                }},
                SECRET.getBytes()
        );
    }

    /**
     * 解析token
     * @param token
     * @return 返回用户信息
     */
    public static User parseToken(String token) {
        JWTPayload payload = JWTUtil.parseToken(token).getPayload();
        return User.builder()
                .username((String) payload.getClaim("username"))
                .build();
    }
}
