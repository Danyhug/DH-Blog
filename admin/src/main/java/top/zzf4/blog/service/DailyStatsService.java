package top.zzf4.blog.service;

import top.zzf4.blog.entity.model.DailyStats;

import java.util.Date;

public interface DailyStatsService{
    // 每日0点自动计算数据
    void daily();

    // 计算文章总数
    Integer countArticle();

    // 计算标签总数
    Integer countTag();

    // 计算评论总数
    Integer countComment();

    // 计算访问总数
    Integer countVisit();

    // 获取目录总数
    Integer countCategory();

    // 获取某天的数据
    DailyStats getDailyStats(Date date);
}
