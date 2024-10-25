package top.zzf4.blog.controller;

import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import lombok.extern.log4j.Log4j2;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.multipart.MultipartFile;
import top.zzf4.blog.entity.AjaxResult;
import top.zzf4.blog.entity.dto.ArticleInsertDTO;
import top.zzf4.blog.entity.dto.ArticlePageDTO;
import top.zzf4.blog.entity.dto.ArticleUpdateDTO;
import top.zzf4.blog.entity.dto.TagInsertDTO;
import top.zzf4.blog.entity.model.Articles;
import top.zzf4.blog.entity.model.Category;
import top.zzf4.blog.entity.vo.PageResult;
import top.zzf4.blog.service.AdminService;
import top.zzf4.blog.service.ArticleService;

import java.io.File;
import java.io.IOException;
import java.sql.SQLIntegrityConstraintViolationException;
import java.util.Objects;
import java.util.UUID;

@Tag(name = "后台管理控制器")
@Log4j2
@RestController
@RequestMapping("/admin")
public class AdminController {
    @Autowired
    private AdminService adminService;
    @Autowired
    private ArticleService service;

    @Value("${upload.path}")
    private String uploadPath;

    /**
     * 新增文章
     * @param article 文章类型
     */
    @Operation(summary = "新增文章")
    @PostMapping("/article")
    public AjaxResult<Void> save(@RequestBody ArticleInsertDTO article) {
        log.info("保存文章 {}", article);
        service.saveArticle(article);
        return AjaxResult.success();
    }

    /**
     * 更新文章
     * @param articleUpdate 文章数据
     */
    @Operation(summary = "更新文章")
    @PutMapping("/article")
    public AjaxResult<Void> update(@RequestBody ArticleUpdateDTO articleUpdate) {
        service.updateArticle(articleUpdate);
        return AjaxResult.success();
    }

    /**
     * 文件上传
     */
    @Operation(summary = "文件上传")
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

    // ******************** 标签相关 ********************

    /**
     * 新增文章标签
     * @param tagInsertDTO 标签数据
     */
    @Operation(summary = "新增文章标签")
    @PostMapping("/tag")
    public AjaxResult<Void> saveTag(@RequestBody TagInsertDTO tagInsertDTO) throws SQLIntegrityConstraintViolationException {
        service.saveTag(tagInsertDTO);
        return AjaxResult.success();
    }

    /**
     * 更新标签信息
     */
    @Operation(summary = "更新标签信息")
    @PutMapping("/tag")
    public AjaxResult<Void> updateTag(@RequestBody top.zzf4.blog.entity.model.Tag tag) {
        service.updateTag(tag);
        return AjaxResult.success();
    }

    /**
     * 删除标签信息
     * @param id 标签id
     */
    @Operation(summary = "删除标签信息")
    @DeleteMapping("/tag/{id}")
    public AjaxResult<String> deleteTag(@PathVariable String id) {
        service.deleteTag(id);
        return AjaxResult.success("已删除标签");
    }

    // ******************** 分类相关 ********************

    /**
     * 新增分类
     */
    @Operation(summary = "新增分类")
    @PostMapping("/category")
    public AjaxResult<Void> saveCategory(@RequestBody Category category) throws SQLIntegrityConstraintViolationException {
        // 保存分类
        service.saveCategory(category);
        // 保存分类默认标签
        service.saveCategoryDefaultTags(category.getId(), category.getTagIds());
        return AjaxResult.success();
    }

    /**
     * 根据id查询信息
     * @param id 文章id
     */
    @Operation(summary = "根据id查询信息")
    @GetMapping("/category/{id}")
    public AjaxResult<Category> getCategoryBySlug(@PathVariable String id) {
        return AjaxResult.success(service.getCategoryById(id));
    }

    /**
     * 更改分类信息
     * @param category 分类信息
     */
    @Operation(summary = "更改分类信息")
    @PutMapping("/category")
    public AjaxResult<Void> updateCategory(@RequestBody Category category) {
        service.updateCategory(category);
        return AjaxResult.success();
    }

    /**
     * 删除分类
     * @param id 分类id
     */
    @Operation(summary = "删除分类")
    @DeleteMapping("/category/{id}")
    public AjaxResult<String> deleteCategory(@PathVariable String id) {
        service.deleteCategory(id);
        return AjaxResult.success("已删除分类");
    }

    /**
     * 分页查询
     */
    @Operation(summary = "分页查询")
    @PostMapping("/article/list")
    public AjaxResult<PageResult<Articles>> getPage(@RequestBody ArticlePageDTO articlePage) {
        log.info("分页查询 {}", articlePage);
        return AjaxResult.success(adminService.getArticleList(articlePage.getPageSize(), articlePage.getPageNum()));
    }
}
