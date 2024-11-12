package top.zzf4.blog.config;

import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import lombok.NonNull;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;
import org.springframework.web.servlet.HandlerInterceptor;
import top.zzf4.blog.service.Impl.AccessLogImpl;

// 博客记录日志相关
@Component
public class AccessLogInterceptor implements HandlerInterceptor {
    @Autowired
    private AccessLogImpl accessLog;

    @Override
    public boolean preHandle(HttpServletRequest request, @NonNull HttpServletResponse response, @NonNull Object handler)
    {
        accessLog.addAccessLog(request);
        return true;
    }
}
