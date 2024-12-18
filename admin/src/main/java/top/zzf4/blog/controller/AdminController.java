package top.zzf4.blog.controller;

import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import jakarta.servlet.http.HttpServletRequest;
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
import top.zzf4.blog.entity.model.Comment;
import top.zzf4.blog.entity.vo.PageResult;
import top.zzf4.blog.service.AdminService;
import top.zzf4.blog.service.ArticleService;
import top.zzf4.blog.service.CommentService;
import top.zzf4.blog.utils.Tools;

import java.io.File;
import java.io.IOException;
import java.sql.SQLIntegrityConstraintViolationException;
import java.util.List;
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
    @Autowired
    private CommentService commentService;

    @Value("${upload.path}")
    private String uploadPath;

    /**
     * 获取文章详情
     * @param id 文章id
     */
    @Operation(summary = "获取文章详情")
    @GetMapping("/article/{id}")
    public AjaxResult<Articles> detail(@PathVariable String id) {
        log.info("获取文章详情 {}", id);
        Articles articleById = service.getArticleById(Long.valueOf(id));

        return AjaxResult.success(articleById);
    }

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
        // 保存分类默认标签
        service.saveCategoryDefaultTags(category.getId(), category.getTagIds());
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
     * 根据分类id查询标签id
     */
    @Operation(summary = "根据分类id查询标签id")
    @GetMapping("/category/{id}/tags")
    public AjaxResult<List<Long>> getTagsByCategoryId(@PathVariable String id) {
        return AjaxResult.success(service.getCategoryDefaultTagsById(Long.valueOf(id)));
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

    // ******************** 评论相关 ********************

    /**
     * 查询所有评论
     */
    @Operation(summary = "查询所有评论")
    @GetMapping("/comment/{pageSize}/{pageNum}")
    public AjaxResult<PageResult<Comment>> getComments(@PathVariable int pageSize, @PathVariable int pageNum) {
        return AjaxResult.success(commentService.getCommentList(pageSize, pageNum, null));
    }

    /**
     * 修改评论
     */
    @Operation(summary = "修改评论")
    @PutMapping("/comment")
    public AjaxResult<String> updateComment(@RequestBody Comment comment) {
        commentService.updateById(comment);
        return AjaxResult.success("修改评论成功！");
    }

    /**
     * 回复评论
     */
    @Operation(summary = "回复评论")
    @PostMapping("/comment/reply")
    public AjaxResult<String> addComment(@RequestBody Comment comment, HttpServletRequest request) {
        String ua = Tools.parseUserAgent(request.getHeader("User-Agent"));
        System.out.println(comment);
        commentService.addComment(
            Comment.builder()
                .author("Danyhug")
                .email("danyhug@zzf4.top")
                .content(comment.getContent())
                .articleId(comment.getArticleId())
                .parentId(comment.getParentId())
                .isPublic(true)
                .isAdmin(true)
                .ua(ua)
                .build()
        );
        return AjaxResult.success("已回复");
    }

    /**
     * 删除评论
     */
    @Operation(summary = "删除评论")
    @DeleteMapping("/comment/{id}")
    public AjaxResult<String> deleteComment(@PathVariable String id) {
        commentService.deleteById(id);
        return AjaxResult.success("已删除评论");
    }

    /**
     * 封禁IP
     */
    @Operation(summary = "封禁IP")
    @PostMapping("/ip/ban/{ip}/{status}")
    public AjaxResult<String> banIp(@PathVariable String ip, @PathVariable String status) {
        if (status == null) return AjaxResult.error("状态不能为空");

        adminService.changeBanIpStatus(ip, status.equals("1") ? 0: 1);
        return AjaxResult.success();
    }
}
