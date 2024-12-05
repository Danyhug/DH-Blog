package top.zzf4.blog.config;

import com.baomidou.mybatisplus.core.toolkit.StringUtils;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import lombok.NonNull;
import lombok.extern.log4j.Log4j2;
import org.springframework.web.servlet.HandlerInterceptor;
import top.zzf4.blog.entity.model.User;
import top.zzf4.blog.entity.vo.SendResponseData;
import top.zzf4.blog.utils.JwtUtils;
import top.zzf4.blog.utils.Tools;

import java.io.IOException;

@Log4j2
public class LoginInterceptor implements HandlerInterceptor {
    @Override
    public boolean preHandle(HttpServletRequest request, @NonNull HttpServletResponse response, @NonNull Object handler) throws IOException {
        // 获取请求头的token
        String token = request.getHeader("Authorization");
        if (StringUtils.isBlank(token)) {
            Tools.sendResponse(new SendResponseData(403, "越权访问，非法请求！", response));
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

        Tools.sendResponse(new SendResponseData(403, "登录状态异常！", response));
        return false;
    }
}
