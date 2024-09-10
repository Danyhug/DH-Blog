package top.zzf4.blog.service;

import com.baomidou.mybatisplus.extension.service.IService;
import top.zzf4.blog.entity.dto.ArticleInsertDTO;
import top.zzf4.blog.entity.dto.ArticlePageDTO;
import top.zzf4.blog.entity.dto.ArticleUpdateDTO;
import top.zzf4.blog.entity.dto.TagInsertDTO;
import top.zzf4.blog.entity.model.Articles;
import top.zzf4.blog.entity.model.Category;
import top.zzf4.blog.entity.model.Tag;
import top.zzf4.blog.entity.vo.PageResult;

import java.io.IOException;
import java.sql.SQLIntegrityConstraintViolationException;
import java.util.List;

public interface ArticleService extends IService<Articles> {
    /**
     * 使用id查询文章信息
     * @param id 文章id
     * @return 文章信息
     */
    Articles getArticleById(Long id);

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

    PageResult<Articles> getPage(ArticlePageDTO articlePage);

    /**
     * 查询文章分类
     */
    List<Category> getArticleCategories();

    /**
     * 保存分类
     */
    void saveCategory(Category category) throws SQLIntegrityConstraintViolationException;

    /**
     * 根据id查询分类
     * @param id 分类id
     * @return 分类数据
     */
    Category getCategoryById(String id);

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
     * 返回随机图片
     * @return 图片字节
     */
    byte[] getRandomImage() throws IOException;

    /**
     * 缓存首页的文章缩略信息
     * 从redis中返回 不带内容 的文章基本信息列表，文章按照id倒序排列
     * @return 文章缩略信息列表
     */
    PageResult<Articles> getArticleThumbnail(int pageSize, int currentPage);
}
