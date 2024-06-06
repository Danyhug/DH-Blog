package top.zzf4.blog.service;

import top.zzf4.blog.entity.dto.ArticleInsertDTO;
import top.zzf4.blog.entity.dto.ArticlePageDTO;
import top.zzf4.blog.entity.dto.ArticleUpdateDTO;
import top.zzf4.blog.entity.dto.TagInsertDTO;
import top.zzf4.blog.entity.model.Article;
import top.zzf4.blog.entity.model.Category;
import top.zzf4.blog.entity.model.Tag;
import top.zzf4.blog.entity.vo.PageResult;

import java.sql.SQLIntegrityConstraintViolationException;
import java.util.List;

public interface ArticleService {
    /**
     * 使用id查询文章信息
     * @param id
     * @return 文章信息
     */
    Article getArticleById(Long id);

    /**
     * 保存文章
     * @param articleInsertDTO
     */
    void saveArticle(ArticleInsertDTO articleInsertDTO);

    /**
     * 更新文章
     * @param article
     */
    void updateArticle(ArticleUpdateDTO article);

    /**
     * 删除文章
     * @param id
     */
    void deleteArticle(Long id);

    /**
     * 保存标签
     * @param tagInsertDTO
     */
    void saveTag(TagInsertDTO tagInsertDTO) throws SQLIntegrityConstraintViolationException;

    /**
     * 查询所有标签
     * @return
     */
    List<Tag> getTags();

    PageResult<Article> getPage(ArticlePageDTO articlePage);

    /**
     * 查询文章分类
     * @return
     */
    List<Category> getArticleCategories();

    /**
     * 保存分类
     * @param category
     */
    void saveCategory(Category category) throws SQLIntegrityConstraintViolationException;

    /**
     * 根据id查询分类
     * @param slug
     * @return
     */
    Category getCategoryById(String id);

    /**
     * 更新分类
     * @param category
     */
    void updateCategory(Category category);

    /**
     * 更新标签
     * @param tag
     */
    void updateTag(Tag tag);
}
