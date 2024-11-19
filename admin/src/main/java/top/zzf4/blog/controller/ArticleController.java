package top.zzf4.blog.controller;

import io.swagger.v3.oas.annotations.Operation;
import lombok.extern.log4j.Log4j2;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;
import top.zzf4.blog.aop.Limit;
import top.zzf4.blog.entity.AjaxResult;
import top.zzf4.blog.entity.dto.ArticlePageDTO;
import top.zzf4.blog.entity.model.Articles;
import top.zzf4.blog.entity.model.Category;
import top.zzf4.blog.entity.model.Tag;
import top.zzf4.blog.entity.vo.OverviewCount;
import top.zzf4.blog.entity.vo.PageResult;
import top.zzf4.blog.service.ArticleService;

import java.util.List;

@Log4j2
@RestController
@RequestMapping("/article")
@io.swagger.v3.oas.annotations.tags.Tag(name = "文章管理控制器")
public class ArticleController {
    @Autowired
    private ArticleService service;

    /**
     * 获取文章详情
     * @param id 文章id
     */
    @Limit(num = 5, time = 60)
    @Operation(summary = "获取文章详情")
    @GetMapping("/{id}")
    public AjaxResult<Articles> detail(@PathVariable String id) {
        log.info("获取文章详情 {}", id);
        Articles articleById = service.getArticleById(Long.valueOf(id));

        // 检查是否需要 解密
        if (articleById.getIsLocked()) {
            articleById.setLockPassword(null);
            return AjaxResult.error("加密文章，请输入密码后访问");
        }

        // pv++
        service.pv();
        return AjaxResult.success(articleById);
    }

    // 获取文章标题
    @Operation(summary = "获取文章标题")
    @GetMapping("/title/{id}")
    public AjaxResult<String> getArticleTitleById(@PathVariable String id) {
        log.info("获取文章标题 {}", id);
        String articleTitleById = service.getArticleTitleById(Long.valueOf(id));

        return AjaxResult.success(articleTitleById);
    }

    /**
     * 获取需要解密的文章
     */
    @Limit(num = 2, time = 60)
    @Operation(summary = "获取需要解密的文章")
    @GetMapping("/unlock/{id}/{password}")
    public AjaxResult<Articles> getLockArticle(@PathVariable String id, @PathVariable String password) {
        log.info("获取需要解密的文章 {}", id);
        Articles articleById = service.getArticleById(Long.valueOf(id));

        if (articleById.isUnLock(password)) {
            return AjaxResult.success(articleById);
        }

        return AjaxResult.error("密码错误");
    }

    /**
     * 分页查询
     */
    @Operation(summary = "分页查询")
    @PostMapping("/list")
    public AjaxResult<PageResult<Articles>> getPage(@RequestBody ArticlePageDTO articlePage) {
        log.info("分页查询 {}", articlePage);
        return AjaxResult.success(service.getArticleThumbnail(articlePage.getPageSize(), articlePage.getPageNum()));
    }

    /**
     * 获取总览
     */
    @Limit
    @Operation(summary = "获取总览")
    @GetMapping("/overview")
    public AjaxResult<OverviewCount> getOverview() {
        return AjaxResult.success(service.getOverview());
    }

    /**
     * 查询标签列表
     */
    @Limit
    @Operation(summary = "查询标签列表")
    @GetMapping("/tag")
    public AjaxResult<List<Tag>> getTags() {
        return AjaxResult.success(service.getTags());
    }


    /**
     * 查询分类列表
     */
    @Limit
    @Operation(summary = "查询分类列表")
    @GetMapping("/category")
    public AjaxResult<List<Category>> getArticles() {
        return AjaxResult.success(service.getArticleCategories());
    }
}
