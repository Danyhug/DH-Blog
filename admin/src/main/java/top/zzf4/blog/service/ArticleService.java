package top.zzf4.blog.service;

import top.zzf4.blog.entity.dto.ArticleInsertDto;
import top.zzf4.blog.entity.dto.ArticlePageDTO;
import top.zzf4.blog.entity.dto.ArticleUpdateDTO;
import top.zzf4.blog.entity.dto.TagInsertDTO;
import top.zzf4.blog.entity.model.Article;
import top.zzf4.blog.entity.model.Category;
import top.zzf4.blog.entity.model.Tag;
import top.zzf4.blog.entity.vo.PageResult;

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
     * @param articleInsertDto
     */
    void saveArticle(ArticleInsertDto articleInsertDto);

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
    void saveTag(TagInsertDTO tagInsertDTO);

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
}
