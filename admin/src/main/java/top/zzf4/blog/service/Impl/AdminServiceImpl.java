package top.zzf4.blog.service.Impl;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.core.conditions.update.LambdaUpdateWrapper;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import top.zzf4.blog.entity.model.Articles;
import top.zzf4.blog.entity.model.IpStat;
import top.zzf4.blog.entity.vo.PageResult;
import top.zzf4.blog.mapper.ArticleMapper;
import top.zzf4.blog.mapper.IpStatMapper;
import top.zzf4.blog.mapper.TagsMapper;
import top.zzf4.blog.service.AdminService;

import java.util.ArrayList;
import java.util.List;

@Service
public class AdminServiceImpl implements AdminService {

    @Autowired
    private ArticleMapper articleMapper;
    @Autowired
    private TagsMapper tagMapper;
    @Autowired
    private IpStatMapper ipStatMapper;

    @Override
    public PageResult<Articles> getArticleList(int pageSize, int currentPage) {
        PageResult<Articles> result = new PageResult<>();

        // 获取所有文章的基本信息
        List<Articles> articles = new ArrayList<>(
                articleMapper.selectList(new LambdaQueryWrapper<Articles>().orderByDesc(Articles::getId))
        );

        // 获取文章的所有标签
        for (Articles article: articles) {
            article.setTags(tagMapper.getTagsByArticleId(article.getId()));
        }

        result.setList(articles);
        result.setCurr((long) currentPage);
        result.setTotal((long) articles.size());
        return result;
    }

    @Override
    public void banIp(String ip) {
        ipStatMapper.update(new LambdaUpdateWrapper<IpStat>().eq(IpStat::getIpAddress, ip).set(IpStat::getBanStatus, 1));
    }

    @Override
    public Boolean isBanned(String ip) {
        if (ip == null) return false;

        IpStat ipStat = ipStatMapper.selectOne(new LambdaQueryWrapper<IpStat>().eq(IpStat::getIpAddress, ip));
        return ipStat != null && ipStat.getBanStatus() == 1;
    }
}
