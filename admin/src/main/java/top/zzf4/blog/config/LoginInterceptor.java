package top.zzf4.blog.config;

import cn.hutool.jwt.JWTException;
import com.baomidou.mybatisplus.core.toolkit.StringUtils;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import lombok.NonNull;
import lombok.extern.log4j.Log4j2;
import org.springframework.web.servlet.HandlerInterceptor;
import top.zzf4.blog.entity.model.User;
import top.zzf4.blog.utils.JwtUtils;

import java.io.IOException;

@Log4j2
public class LoginInterceptor implements HandlerInterceptor {
    @Override
    public boolean preHandle(HttpServletRequest request, @NonNull HttpServletResponse response, @NonNull Object handler) throws IOException {
        // 获取请求头的token
        String token = request.getHeader("Authorization");
        if (StringUtils.isBlank(token)) {
            sendUnauthorizedResponse(response);
            return false;
        }

        try {
            User user = JwtUtils.parseToken(token);
            // 如果获取到了用户并且用户名不为空，则放行
            if (user != null && StringUtils.isNotEmpty(user.getUsername())) {
                return true;
            }
        } catch (Exception e) {
            // 记录异常日志
            log.error("Token解析失败: {}", e.getMessage(), e);
        }

        sendUnauthorizedResponse(response);
        return false;
    }

    private void sendUnauthorizedResponse(HttpServletResponse response) throws IOException {
        response.setStatus(401);
        response.setContentType("text/plain;charset=utf-8");
        response.getWriter().write("非法请求");
    }
}
