package top.zzf4.blog.controller;

import lombok.extern.log4j.Log4j2;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;
import top.zzf4.blog.entity.AjaxResult;
import top.zzf4.blog.entity.dto.ArticleInsertDto;
import top.zzf4.blog.entity.dto.ArticlePageDTO;
import top.zzf4.blog.entity.dto.ArticleUpdateDTO;
import top.zzf4.blog.entity.dto.TagInsertDTO;
import top.zzf4.blog.entity.model.Article;
import top.zzf4.blog.entity.model.Category;
import top.zzf4.blog.entity.model.Tag;
import top.zzf4.blog.entity.vo.PageResult;
import top.zzf4.blog.service.ArticleService;

import java.util.List;

@RestController
@Log4j2
@CrossOrigin
@RequestMapping("/article")
public class ArticleController {
    @Autowired
    private ArticleService service;

    /**
     * 获取文章详情
     * @param id
     * @return
     */
    @GetMapping("/{id}")
    public AjaxResult<Article> detail(@PathVariable String id) {
        log.info("获取文章详情 {}", id);
        return AjaxResult.success(
                service.getArticleById(Long.valueOf(id))
        );
    }

    /**
     * 新增文章
     * @param article
     * @return
     */
    @PostMapping
    public AjaxResult<Void> save(@RequestBody ArticleInsertDto article) {
        log.info("保存文章 {}", article);
        service.saveArticle(article);
        return AjaxResult.success();
    }

    /**
     * 更新文章
     * @param articleUpdate
     * @return
     */
    @PutMapping
    public AjaxResult<Void> update(@RequestBody ArticleUpdateDTO articleUpdate) {
        service.updateArticle(articleUpdate);
        return AjaxResult.success();
    }

    /**
     * 分页查询
     * @return
     */
    @GetMapping
    public AjaxResult<PageResult<Article>> getPage(@RequestBody ArticlePageDTO articlePage) {
        log.info("分页查询 {}", articlePage);
        return AjaxResult.success(service.getPage(articlePage));
    }

    /**
     * 新增文章标签
     * @param tagInsertDTO
     * @return
     */
    @PostMapping("/tag")
    public AjaxResult<Void> saveTag(@RequestBody TagInsertDTO tagInsertDTO) {
        service.saveTag(tagInsertDTO);
        return AjaxResult.success();
    }

    /**
     * 查询标签列表
     * @return
     */
    @GetMapping("/tag")
    public AjaxResult<List<Tag>> getTags() {
        return AjaxResult.success(service.getTags());
    }

    /**
     * 查询分类列表
     */
    @GetMapping("/category")
    public AjaxResult<List<Category>> getArticles() {
        return AjaxResult.success(service.getArticleCategories());
    }
}
