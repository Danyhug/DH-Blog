package top.zzf4.blog.aop;

import jakarta.servlet.http.HttpServletRequest;
import org.aspectj.lang.ProceedingJoinPoint;
import org.aspectj.lang.annotation.Around;
import org.aspectj.lang.annotation.Aspect;
import org.aspectj.lang.reflect.MethodSignature;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;
import org.springframework.web.context.request.RequestContextHolder;
import org.springframework.web.context.request.ServletRequestAttributes;
import top.zzf4.blog.constant.RedisConstant;
import top.zzf4.blog.utils.RedisCacheUtils;

import java.util.concurrent.TimeUnit;

@Aspect
@Component
public class LimitAop {

    @Autowired
    private RedisCacheUtils redisCacheUtils;

    @Around("@annotation(limit)")
    public Object rateLimiter(ProceedingJoinPoint joinPoint, Limit limit) throws Throwable {
        // 获取当前请求的 HttpServletRequest 对象
        ServletRequestAttributes attributes = (ServletRequestAttributes) RequestContextHolder.getRequestAttributes();
        HttpServletRequest request = attributes.getRequest();

        // 获取客户端 IP 地址
        String clientIp = getClientIp(request);
        // 打印或处理客户端 IP 地址
        System.out.println("Client IP: " + clientIp);

        MethodSignature methodSignature = (MethodSignature) joinPoint.getSignature();
        String methodName = methodSignature.getName();

        String key = RedisConstant.CACHE_IP + methodName + ":" + clientIp;
        // 不存在ip的情况
        if (redisCacheUtils.hasNullKey(key)) {
            // 缓存客户端 IP 地址
            redisCacheUtils.set(key, 1);
            // 每10秒钟限制访问接口10次
            redisCacheUtils.setExpire(key, limit.time(), TimeUnit.SECONDS);
        } else {
            // 获取当前 IP 地址的访问次数
            Integer count = (Integer) redisCacheUtils.get(key);
            // 判断是否超过限制次数
            if (count >= limit.num()) {
                // 说明超过次数，抛出异常
                throw new RuntimeException(limit.msg());
            } else {
                // 累加访问次数
                redisCacheUtils.incr(key);
            }
        }

        // 继续执行目标方法
        return joinPoint.proceed();
    }

    /**
     * 获取客户端真实 IP 地址
     * @param request 当前请求
     * @return 客户端 IP 地址
     */
    private String getClientIp(HttpServletRequest request) {
        String ip = request.getHeader("X-Forwarded-For");
        if (ip == null || ip.length() == 0 || "unknown".equalsIgnoreCase(ip)) {
            ip = request.getHeader("Proxy-Client-IP");
        }
        if (ip == null || ip.length() == 0 || "unknown".equalsIgnoreCase(ip)) {
            ip = request.getHeader("WL-Proxy-Client-IP");
        }
        if (ip == null || ip.length() == 0 || "unknown".equalsIgnoreCase(ip)) {
            ip = request.getHeader("HTTP_CLIENT_IP");
        }
        if (ip == null || ip.length() == 0 || "unknown".equalsIgnoreCase(ip)) {
            ip = request.getHeader("HTTP_X_FORWARDED_FOR");
        }
        if (ip == null || ip.length() == 0 || "unknown".equalsIgnoreCase(ip)) {
            ip = request.getRemoteAddr();
        }

        // 将 IPv6 的本地回环地址转换为 IPv4 的本地回环地址
        if (ip.equals("0:0:0:0:0:0:0:1") || ip.equals("::1")) {
            ip = "127.0.0.1";
        }

        return ip;
    }
}
