package top.zzf4.blog.utils;

import jakarta.servlet.http.HttpServletRequest;

public class Tools {
    /**
     * 获取客户端真实 IP 地址
     * @param request 当前请求
     * @return 客户端 IP 地址
     */
    public static String getClientIp(HttpServletRequest request) {
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
