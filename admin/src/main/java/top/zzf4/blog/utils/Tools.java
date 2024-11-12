package top.zzf4.blog.utils;

import cn.hutool.core.bean.BeanUtil;
import cn.hutool.http.HttpUtil;
import cn.hutool.json.JSONUtil;
import jakarta.servlet.http.HttpServletRequest;
import lombok.extern.log4j.Log4j2;
import top.zzf4.blog.utils.ip.Csdn;

import java.util.HashMap;
import java.util.Map;

/**
 * { "code": 200, "msg": "success", "data": { "address": "中国 河北 石家庄 联通", "ip": "218.12.18.209" } }
 */
@Log4j2
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

    /**
     * 根据ua获取系统和浏览器
     */
    public static String parseUserAgent(String userAgentString) {
        String os = "";
        String browser = "";

        // 解析操作系统
        if (userAgentString.contains("Windows NT 10.0")) {
            os = "Windows 10";
        } else if (userAgentString.contains("Windows NT 6.3")) {
            os = "Windows 8.1";
        } else if (userAgentString.contains("Windows NT 6.2")) {
            os = "Windows 8";
        } else if (userAgentString.contains("Windows NT 6.1")) {
            os = "Windows 7";
        } else if (userAgentString.contains("Macintosh") || userAgentString.contains("Mac OS X")) {
            os = "Mac OS";
        } else if (userAgentString.contains("Android")) {
            os = "Android";
        } else if (userAgentString.contains("iPhone") || userAgentString.contains("iPad") || userAgentString.contains("iPod")) {
            os = "iOS";
        } else if (userAgentString.contains("HarmonyOS")) {
            os = "鸿蒙";
        } else if (userAgentString.contains("Linux")) {
            os = "Linux";
        }

        // 解析浏览器
        if (userAgentString.contains("Edg/")) {
            browser = "Edge " + userAgentString.split("Edg/")[1].split(" ")[0];
        } else if (userAgentString.contains("Chrome/")) {
            browser = "Chrome " + userAgentString.split("Chrome/")[1].split(" ")[0];
        } else if (userAgentString.contains("Firefox/")) {
            browser = "Firefox " + userAgentString.split("Firefox/")[1].split(" ")[0];
        } else if (userAgentString.contains("Safari/")) {
            browser = "Safari " + userAgentString.split("Version/")[1].split(" ")[0];
        }

        return os + "; " + browser;
    }

    /**
     * 获取 IP 地址对应的城市
     * @param ip
     * @return
     */
    public static String getIpCity(String ip) {
        /**
         * 免费接口
         * https://searchplugin.csdn.net/api/v1/ip/get?ip=123.123.123.123
         * http://ip-api.com/json/123.123.123.123?lang=zh-CN
         * https://whois.pconline.com.cn/ipJson.jsp?ip=123.123.123.123&json=true
         * https://sp0.baidu.com/8aQDcjqpAAV3otqbppnN2DJv/api.php?query=123.123.123.123&co=&resource_id=6006&oe=utf8
         */
        String url = "https://searchplugin.csdn.net/api/v1/ip/get?ip=" + ip;
        // 伪装为浏览器发送请求
        // 创建一个Map对象来设置请求头
        Map<String, String> headers = new HashMap<>();
        headers.put("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36");
        // 发送GET请求并获取响应
        String responseString = HttpUtil.createGet(url).addHeaders(headers).execute().body();
        log.info("IP 地址查询结果: {}", responseString);

        // 转为 JSON 对象
        Csdn bean = JSONUtil.toBean(responseString, Csdn.class);
        if (bean.getCode() == 200) {
            return bean.getData().getAddress();
        }

        return "";
    }
}
