package top.zzf4.blog.schedule;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.scheduling.annotation.Async;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;
import top.zzf4.blog.service.DailyStatsService;

@Component
public class AutoTask {
    @Autowired
    private DailyStatsService dailyStatsService;

    /**
     * 每日自动任务
     */
    @Async
    @Scheduled(cron = "0 0 0 * * ? ")
    public void run() {
        // 统计每日访问数
        dailyStatsService.start();
    }
}
