package top.zzf4.blog.service.Impl;

import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import com.baomidou.mybatisplus.core.conditions.update.LambdaUpdateWrapper;
import jakarta.servlet.http.HttpServletRequest;
import lombok.extern.log4j.Log4j2;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import top.zzf4.blog.entity.model.AccessLog;
import top.zzf4.blog.entity.model.IpStat;
import top.zzf4.blog.mapper.AccessLogMapper;
import top.zzf4.blog.mapper.IpStatMapper;
import top.zzf4.blog.utils.Tools;

@Log4j2
@Service
public class AccessLogImpl {
    @Autowired
    private AccessLogMapper accessLogMapper;

    @Autowired
    private IpStatMapper ipStatMapper;

    /**
     * 向数据库中插入访问日志
     */
    public void addAccessLog(HttpServletRequest request) {
        AccessLog accessLog = new AccessLog();
        // 获取客户端 IP 地址
        accessLog.setIpAddress(Tools.getClientIp(request));
        // 获取客户端的系统和浏览器版本
        accessLog.setUserAgent(Tools.parseUserAgent(request.getHeader("User-Agent")));

        accessLog.setRequestUrl(request.getRequestURI());
        accessLogMapper.insert(accessLog);
        log.info("访问日志插入成功：{}", accessLog);

        IpStat ipStat = ipStatMapper.selectOne(new QueryWrapper<IpStat>().eq("ip_address", accessLog.getIpAddress()));
        // 查询该ip是否存在
        if (ipStat != null) {
            // 存在，让访问次数增加即可
            ipStatMapper.update(
                    IpStat.builder().accessCount(ipStat.getAccessCount() + 1).build(),
                    new LambdaUpdateWrapper<IpStat>().eq(IpStat::getIpAddress, ipStat.getIpAddress())
            );
        } else {
            // 否则插入新的
            ipStatMapper.insert(
                    IpStat.builder().ipAddress(accessLog.getIpAddress())
                            .accessCount(1).bannedCount(0).city(Tools.getIpCity(accessLog.getIpAddress()))
                            .build()
            );
        }
        log.info("IP 统计插入成功：{}", ipStat);
    }
}
