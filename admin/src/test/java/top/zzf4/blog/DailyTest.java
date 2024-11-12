package top.zzf4.blog;

import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import top.zzf4.blog.service.DailyStatsService;

@SpringBootTest
public class DailyTest {
    @Autowired
    private DailyStatsService dailyStatsService;

    @Test
    public void init() {
        // dailyStatsService.daily();
    }
}
