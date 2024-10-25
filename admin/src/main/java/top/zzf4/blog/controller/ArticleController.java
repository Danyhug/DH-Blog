package top.zzf4.blog.controller;

import io.swagger.v3.oas.annotations.Operation;
import lombok.extern.log4j.Log4j2;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import top.zzf4.blog.entity.AjaxResult;
import top.zzf4.blog.entity.dto.ArticlePageDTO;
import top.zzf4.blog.entity.model.Articles;
import top.zzf4.blog.entity.model.Category;
import top.zzf4.blog.entity.model.Tag;
import top.zzf4.blog.entity.vo.PageResult;
import top.zzf4.blog.service.ArticleService;

import java.io.IOException;
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
    @Operation(summary = "获取文章详情")
    @GetMapping("/{id}")
    public AjaxResult<Articles> detail(@PathVariable String id) {
        log.info("获取文章详情 {}", id);
        Articles articleById = service.getArticleById(Long.valueOf(id));

        // 检查是否需要 解密
        if (articleById.getIsLocked()) {
            return AjaxResult.error("加密文章，请输入密码后访问");
        }

        return AjaxResult.success(articleById);
    }

    /**
     * 获取需要解密的文章
     */
    @Operation(summary = "获取需要解密的文章")
    @PostMapping("/unlock")
    public AjaxResult<Articles> getLockArticle(@RequestBody String id, @RequestBody String password) {
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
     * 获取随机图片
     */
    @Operation(summary = "为首页返回随机图片")
    @GetMapping("/image/random")
    public ResponseEntity<byte[]> getRandomImage() throws IOException {
        // 设置响应头
        HttpHeaders headers = new HttpHeaders();
        headers.setContentType(MediaType.IMAGE_JPEG);
        // headers.set("Cache-Control", "no-cache, no-store, must-revalidate");
        // headers.set("Pragma", "no-cache");
        // headers.set("Expires", "0");
        return new ResponseEntity<>(service.getRandomImage(), headers, HttpStatus.OK);
    }

    /**
     * 查询标签列表
     */
    @Operation(summary = "查询标签列表")
    @GetMapping("/tag")
    public AjaxResult<List<Tag>> getTags() {
        return AjaxResult.success(service.getTags());
    }


    /**
     * 查询分类列表
     */
    @Operation(summary = "查询分类列表")
    @GetMapping("/category")
    public AjaxResult<List<Category>> getArticles() {
        return AjaxResult.success(service.getArticleCategories());
    }
}
