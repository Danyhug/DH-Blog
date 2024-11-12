package top.zzf4.blog.service.Impl;

import cn.hutool.core.date.DateTime;
import cn.hutool.core.date.DateUtil;
import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import lombok.extern.log4j.Log4j2;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import top.zzf4.blog.constant.RedisConstant;
import top.zzf4.blog.entity.model.DailyStats;
import top.zzf4.blog.mapper.ArticleMapper;
import top.zzf4.blog.mapper.CategoriesMapper;
import top.zzf4.blog.mapper.DailyStatsMapper;
import top.zzf4.blog.mapper.TagsMapper;
import top.zzf4.blog.service.DailyStatsService;
import top.zzf4.blog.utils.RedisCacheUtils;

import java.util.Date;

@Service
@Log4j2
public class DailyStatsServiceImpl implements DailyStatsService {
    @Autowired
    private ArticleMapper articleMapper;
    @Autowired
    private DailyStatsMapper dailyStatsMapper;
    @Autowired
    private TagsMapper tagsMapper;
    @Autowired
    private CategoriesMapper categoriesMapper;
    @Autowired
    private RedisCacheUtils RedisUtil;

    @Override
    public void start() {
        // 获取前一日日期
        DateTime dateTime = DateUtil.offsetDay(new Date(), -1);
        DailyStats build = DailyStats.builder()
                .date(dateTime)
                .articleCount(countArticle())
                .commentCount(countComment())
                .tagCount(countTag())
                .visitCount(countVisit())
                .build();
        log.info("{} 数据 {}", dateTime.toString(), build);
        dailyStatsMapper.insert(build);
    }

    @Override
    public Integer countArticle() {
        return articleMapper.selectCount(new LambdaQueryWrapper<>()).intValue();
    }

    @Override
    public Integer countTag() {
        return tagsMapper.selectCount(new LambdaQueryWrapper<>()).intValue();
    }

    @Override
    public Integer countComment() {
        return 0;
    }

    @Override
    public Integer countVisit() {
        // 获取前一日的pv
        String yesterday = DateUtil.format(DateUtil.offsetDay(new Date(), -1), "yyyy-MM-dd");
        String key = RedisConstant.CACHE_DAILY_PV + yesterday;
        Integer yesterdayPv = (Integer) RedisUtil.get(key);
        // 删除缓存
        RedisUtil.delete(key);
        return yesterdayPv;
    }

    @Override
    public Integer countCategory() {
        return categoriesMapper.selectCount(new LambdaQueryWrapper<>()).intValue();
    }

    @Override
    public DailyStats getDailyStats(Date date) {
        return null;
    }
}
