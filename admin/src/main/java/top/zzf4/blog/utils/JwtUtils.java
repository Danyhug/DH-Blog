package top.zzf4.blog.utils;

import cn.hutool.crypto.digest.BCrypt;
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
                    // 添加过期时间为7天
                    put("expire", System.currentTimeMillis() + 7 * 24 * 60 * 60 * 1000);
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
        if (payload == null) {
            throw new RuntimeException("token无效");
        }

        Long expire = (Long) payload.getClaim("expire");
        if (expire < System.currentTimeMillis()) {
            throw new RuntimeException("token 已过期");
        }

        return User.builder()
                .username((String) payload.getClaim("username"))
                .build();
    }

    public static String encodeByBCrypt(String str) {
        return BCrypt.hashpw(str, BCrypt.gensalt());
    }

    public static boolean verifyByBCrypt(String str, String hash) {
        return BCrypt.checkpw(str, hash);
    }
}
