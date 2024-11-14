package top.zzf4.blog.controller;

import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import lombok.extern.log4j.Log4j2;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import top.zzf4.blog.entity.AjaxResult;
import top.zzf4.blog.entity.model.IpStat;
import top.zzf4.blog.entity.vo.PageResult;
import top.zzf4.blog.service.Impl.AccessLogImpl;

import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.Date;

@Tag(name = "日志控制器")
@Log4j2
@RestController
@RequestMapping("/log")
public class LogController {
    @Autowired
    private AccessLogImpl accessLog;


    /**
     * 获取指定日期范围内的访问记录
     *
     * @param page     当前页码
     * @param pageSize 每页大小
     * @param startDate 开始日期
     * @param endDate   结束日期
     * @return 分页结果
     */
    @Operation(summary = "获取预览页的访问记录")
    @GetMapping("/overview/visitLog")
    public AjaxResult<PageResult<IpStat>> getOverAccessLog(@RequestParam int page, @RequestParam int pageSize,
                                               @RequestParam String startDate, @RequestParam String endDate) {
        try {
            SimpleDateFormat sdf = new SimpleDateFormat("yyyy-MM-dd");
            Date start = sdf.parse(startDate);
            Date end = sdf.parse(endDate);
            return AjaxResult.success(accessLog.getOverAccessLog(page, pageSize, start, end));
        } catch (ParseException e) {
            throw new RuntimeException("日期格式不正确，请使用 yyyy-MM-dd 格式", e);
        }
    }
}
