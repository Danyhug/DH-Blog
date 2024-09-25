package top.zzf4.blog.controller;

import io.swagger.v3.oas.annotations.Operation;
import lombok.extern.log4j.Log4j2;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.multipart.MultipartFile;
import top.zzf4.blog.entity.AjaxResult;
import top.zzf4.blog.entity.dto.ArticleInsertDTO;
import top.zzf4.blog.entity.dto.ArticlePageDTO;
import top.zzf4.blog.entity.dto.ArticleUpdateDTO;
import top.zzf4.blog.entity.dto.TagInsertDTO;
import top.zzf4.blog.entity.model.Articles;
import top.zzf4.blog.entity.model.Category;
import top.zzf4.blog.entity.model.Tag;
import top.zzf4.blog.entity.vo.PageResult;
import top.zzf4.blog.service.ArticleService;

import java.io.File;
import java.io.IOException;
import java.sql.SQLIntegrityConstraintViolationException;
import java.util.List;
import java.util.Objects;
import java.util.UUID;

@Log4j2
@CrossOrigin
@RestController
@RequestMapping("/article")
@io.swagger.v3.oas.annotations.tags.Tag(name = "文章管理控制器")
public class ArticleController {
    @Autowired
    private ArticleService service;

    @Value("${upload.path}")
    private String uploadPath;

    /**
     * 获取文章详情
     * @param id 文章id
     */
    @Operation(description = "获取文章详情")
    @GetMapping("/{id}")
    public AjaxResult<Articles> detail(@PathVariable String id) {
        log.info("获取文章详情 {}", id);
        return AjaxResult.success(
                service.getArticleById(Long.valueOf(id))
        );
    }

    /**
     * 新增文章
     * @param article 文章类型
     */
    @Operation(description = "新增文章")
    @PostMapping
    public AjaxResult<Void> save(@RequestBody ArticleInsertDTO article) {
        log.info("保存文章 {}", article);
        service.saveArticle(article);
        return AjaxResult.success();
    }

    /**
     * 更新文章
     * @param articleUpdate 文章数据
     */
    @Operation(description = "更新文章")
    @PutMapping
    public AjaxResult<Void> update(@RequestBody ArticleUpdateDTO articleUpdate) {
        service.updateArticle(articleUpdate);
        return AjaxResult.success();
    }

    /**
     * 分页查询
     */
    @Operation(description = "分页查询")
    @PostMapping("/list")
    public AjaxResult<PageResult<Articles>> getPage(@RequestBody ArticlePageDTO articlePage) {
        log.info("分页查询 {}", articlePage);
        return AjaxResult.success(service.getArticleThumbnail(articlePage.getPageSize(), articlePage.getPageNum()));
    }

    /**
     * 文件上传
     */
    @Operation(description = "文件上传")
    @PostMapping("/upload")
    public AjaxResult<String> upload(@RequestParam("file") MultipartFile file) throws IOException {
        log.info("上传文件 {} {}", file, uploadPath);
        String originalFilename = file.getOriginalFilename();
        // 截取文件后缀
        String extension = Objects.requireNonNull(originalFilename).substring(originalFilename.lastIndexOf("."));
        // 使用UUID作为文件名
        String objectName = UUID.randomUUID() + extension;

        file.transferTo(new File(uploadPath + objectName));
        return AjaxResult.success(objectName);
    }

    /**
     * 获取随机图片
     */
    @Operation(description = "为首页返回随机图片")
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

    // ******************** 标签相关 ********************

    /**
     * 新增文章标签
     * @param tagInsertDTO 标签数据
     */
    @Operation(description = "新增文章标签")
    @PostMapping("/tag")
    public AjaxResult<Void> saveTag(@RequestBody TagInsertDTO tagInsertDTO) throws SQLIntegrityConstraintViolationException {
        service.saveTag(tagInsertDTO);
        return AjaxResult.success();
    }

    /**
     * 查询标签列表
     */
    @Operation(description = "查询标签列表")
    @GetMapping("/tag")
    public AjaxResult<List<Tag>> getTags() {
        return AjaxResult.success(service.getTags());
    }

    /**
     * 更新标签信息
     */
    @PutMapping("/tag")
    public AjaxResult<Void> updateTag(@RequestBody Tag tag) {
        service.updateTag(tag);
        return AjaxResult.success();
    }

    /**
     * 删除标签信息
     * @param id 标签id
     */
    @DeleteMapping("/tag/{id}")
    public AjaxResult<String> deleteTag(@PathVariable String id) {
        service.deleteTag(id);
        return AjaxResult.success("已删除标签");
    }

    // ******************** 分类相关 ********************

    /**
     * 查询分类列表
     */
    @GetMapping("/category")
    public AjaxResult<List<Category>> getArticles() {
        return AjaxResult.success(service.getArticleCategories());
    }

    /**
     * 新增分类
     */
    @PostMapping("/category")
    public AjaxResult<Void> saveCategory(@RequestBody Category category) throws SQLIntegrityConstraintViolationException {
        service.saveCategory(category);
        return AjaxResult.success();
    }

    /**
     * 根据id查询信息
     * @param id 文章id
     */
    @GetMapping("/category/{id}")
    public AjaxResult<Category> getCategoryBySlug(@PathVariable String id) {
        return AjaxResult.success(service.getCategoryById(id));
    }

    /**
     * 更改分类信息
     * @param category 分类信息
     */
    @PutMapping("/category")
    public AjaxResult<Void> updateCategory(@RequestBody Category category) {
        service.updateCategory(category);
        return AjaxResult.success();
    }

    /**
     * 删除分类
     * @param id 分类id
     */
    @DeleteMapping("/category/{id}")
    public AjaxResult<String> deleteCategory(@PathVariable String id) {
        service.deleteCategory(id);
        return AjaxResult.success("已删除分类");
    }
}
