package top.zzf4.blog.utils;

import cn.hutool.core.bean.BeanUtil;
import cn.hutool.http.HttpUtil;
import cn.hutool.json.JSONUtil;
import jakarta.servlet.http.HttpServletRequest;
import top.zzf4.blog.utils.ip.Csdn;

/**
 * { "code": 200, "msg": "success", "data": { "address": "中国 河北 石家庄 联通", "ip": "218.12.18.209" } }
 */
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
        // 响应的字符串
        String responseString = HttpUtil.get(url);
        // 转为 JSON 对象
        Csdn bean = BeanUtil.toBean(responseString, Csdn.class);

        if (bean.getCode() == 200) {
            return bean.getData().getAddress();
        }

        return "";
    }
}
