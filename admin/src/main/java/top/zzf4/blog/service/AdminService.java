package top.zzf4.blog.service;

import top.zzf4.blog.entity.model.Articles;
import top.zzf4.blog.entity.vo.PageResult;

public interface AdminService {
    /**
     * 获取文章列表
     * @return 文章信息列表
     */
    PageResult<Articles> getArticleList(int pageSize, int currentPage);

    /**
     * 封禁指定IP
     */
    void banIp(String ip);

    /**
     * 查询IP是否被封禁
     */
    Boolean isBanned(String ip);
}
