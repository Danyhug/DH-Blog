package top.zzf4.blog.service.Impl;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import com.baomidou.mybatisplus.core.conditions.update.LambdaUpdateWrapper;
import jakarta.servlet.http.HttpServletRequest;
import lombok.extern.log4j.Log4j2;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import top.zzf4.blog.entity.model.AccessLog;
import top.zzf4.blog.entity.model.IpStat;
import top.zzf4.blog.entity.vo.PageResult;
import top.zzf4.blog.mapper.AccessLogMapper;
import top.zzf4.blog.mapper.IpStatMapper;
import top.zzf4.blog.utils.Tools;

import java.sql.SQLIntegrityConstraintViolationException;
import java.util.*;
import java.util.stream.Collectors;

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
    @Transactional
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
            try {
                ipStatMapper.insert(
                    IpStat.builder().ipAddress(accessLog.getIpAddress())
                            .accessCount(1).bannedCount(0).city(Tools.getIpCity(accessLog.getIpAddress()))
                            .build()
                );
            } catch (Exception ignored){}
        }
        log.info("IP 统计插入成功：{}", ipStat);
    }

    /**
     * 分页获取数据总览页的访问ip访问记录
     */
    public PageResult<IpStat> getOverAccessLog(int page, int pageSize, Date startDate, Date endDate) {
        // 查询指定范围的所有IP
        List<String> allIP = accessLogMapper.selectList(new LambdaQueryWrapper<AccessLog>().select(AccessLog::getIpAddress)
                        .between(AccessLog::getAccessTime, startDate, endDate))
                .stream().map(AccessLog::getIpAddress).toList();

        // 去重
        Set<String> uniqueIPs = new HashSet<>(allIP);

        // 计算每个IP的访问次数
        Map<String, Long> ipAccessCountMap = allIP.stream()
                .collect(Collectors.groupingBy(ip -> ip, Collectors.counting()));

        // 根据IP查询城市信息
        List<IpStat> ipStats = ipStatMapper.selectList(
                new LambdaQueryWrapper<IpStat>().in(IpStat::getIpAddress, uniqueIPs)
        );

        // 构建结果列表并排序
        List<IpStat> result = ipStats.stream().map(ipStat -> {
                    Long accessCount = ipAccessCountMap.get(ipStat.getIpAddress());
                    return IpStat.builder()
                            .ipAddress(ipStat.getIpAddress())
                            .city(ipStat.getCity())
                            .accessCount(accessCount == null ? 0 : accessCount.intValue())
                            .bannedCount(ipStat.getBannedCount())
                            .build();
                }).sorted(Comparator.comparing(IpStat::getAccessCount, Comparator.reverseOrder())
                        .thenComparing(IpStat::getIpAddress))
                .collect(Collectors.toList());

        // 分页处理
        int fromIndex = (page - 1) * pageSize;
        int toIndex = Math.min(fromIndex + pageSize, result.size());

        // 如果 fromIndex 超过了 result 的大小，返回一个空的分页结果
        if (fromIndex >= result.size()) {
            List<IpStat> emptyResult = Collections.emptyList();
            PageResult<IpStat> pageResult = new PageResult<>();
            pageResult.setList(emptyResult);
            pageResult.setTotal((long) result.size());
            return pageResult;
        }

        List<IpStat> paginatedResult = result.subList(fromIndex, toIndex);

        // 返回分页结果
        PageResult<IpStat> pageResult = new PageResult<>();
        pageResult.setList(paginatedResult);
        pageResult.setTotal((long) result.size());
        return pageResult;
    }

}
