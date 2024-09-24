package top.zzf4.blog.service;

import top.zzf4.blog.entity.dto.ArticleInsertDTO;
import top.zzf4.blog.entity.dto.ArticleUpdateDTO;
import top.zzf4.blog.entity.dto.TagInsertDTO;
import top.zzf4.blog.entity.model.Articles;
import top.zzf4.blog.entity.model.Category;
import top.zzf4.blog.entity.model.Tag;
import top.zzf4.blog.entity.vo.PageResult;

import java.sql.SQLIntegrityConstraintViolationException;
import java.util.List;

public interface AdminService {
    /**
     * 保存文章
     */
    void saveArticle(ArticleInsertDTO articleInsertDTO);

    /**
     * 更新文章
     * @param article 文章信息
     */
    void updateArticle(ArticleUpdateDTO article);

    /**
     * 删除文章
     * @param id 文章id
     */
    void deleteArticle(Long id);

    /**
     * 保存标签
     */
    void saveTag(TagInsertDTO tagInsertDTO) throws SQLIntegrityConstraintViolationException;

    /**
     * 查询所有标签
     */
    List<Tag> getTags();

    /**
     * 查询文章分类
     */
    List<Category> getArticleCategories();

    /**
     * 保存分类
     */
    void saveCategory(Category category) throws SQLIntegrityConstraintViolationException;

    /**
     * 删除分类
     * @param id 分类id
     */
    void deleteCategory(String id);

    /**
     * 更新分类
     * @param category 分类信息
     */
    void updateCategory(Category category);

    /**
     * 更新标签
     */
    void updateTag(Tag tag);

    /**
     * 删除标签
     * @param id 标签id
     */
    void deleteTag(String id);

    /**
     * 获取文章列表
     * @return 文章信息列表
     */
    PageResult<Articles> getArticleList(int pageSize, int currentPage);
}
