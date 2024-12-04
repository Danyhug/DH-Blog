package top.zzf4.blog.config;

import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import lombok.NonNull;
import lombok.extern.log4j.Log4j2;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;
import org.springframework.web.servlet.HandlerInterceptor;
import top.zzf4.blog.service.AdminService;
import top.zzf4.blog.utils.Tools;

import java.io.IOException;

/**
 * 封禁拦截器
 */
@Log4j2
@Component
public class BanInterceptor implements HandlerInterceptor {
    @Autowired
    private AdminService adminService;

    @Override
    public boolean preHandle(HttpServletRequest request, @NonNull HttpServletResponse response, @NonNull Object handler) throws IOException {
        try {
            // 获取客户端ip
            String clientIp = Tools.getClientIp(request);
            if (!adminService.isBanned(clientIp)) return true;

        } catch (Exception e) {
            // 记录异常日志
            log.error("ip解析失败: {}", e.getMessage(), e);
        }

        sendUnauthorizedResponse(response);
        return false;
    }

    private void sendUnauthorizedResponse(HttpServletResponse response) throws IOException {
        response.setStatus(403);
        response.setContentType("text/plain;charset=utf-8");
        response.getWriter().write("您的IP已被封禁，禁止访问");
    }
}
