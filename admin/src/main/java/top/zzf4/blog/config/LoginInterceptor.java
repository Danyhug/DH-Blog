package top.zzf4.blog.config;

import cn.hutool.core.util.StrUtil;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import lombok.NonNull;
import org.springframework.web.servlet.HandlerInterceptor;
import top.zzf4.blog.entity.model.User;
import top.zzf4.blog.utils.JwtUtils;

import java.io.IOException;


public class LoginInterceptor implements HandlerInterceptor {
    @Override
    public boolean preHandle(HttpServletRequest request, @NonNull HttpServletResponse response, @NonNull Object handler) throws IOException {
        // 获取请求头的token
        String token = request.getHeader("authorization");
        if (StrUtil.isBlank(token)) {
            response.setStatus(401);
            // 显示文字 非法请求
            response.setContentType("text/plain;charset=utf-8");
            response.getWriter().write("非法请求");
            return false;
        }

        User user = JwtUtils.parseToken(token);
        // 如果获取到了用户并且用户名不为空，则放行
        if (user == null ||user.getUsername().isEmpty()) {
            response.setStatus(401);
            response.setContentType("text/plain;charset=utf-8");
            response.getWriter().write("非法请求");
            return false;
        }

        return true;
    }
}
