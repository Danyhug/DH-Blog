package top.zzf4.blog.service.Impl;

import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import top.zzf4.blog.entity.dto.ArticleInsertDTO;
import top.zzf4.blog.entity.dto.ArticleUpdateDTO;
import top.zzf4.blog.entity.dto.TagInsertDTO;
import top.zzf4.blog.entity.model.Articles;
import top.zzf4.blog.entity.model.Category;
import top.zzf4.blog.entity.model.Tag;
import top.zzf4.blog.entity.vo.PageResult;
import top.zzf4.blog.mapper.ArticleMapper;
import top.zzf4.blog.mapper.TagsMapper;
import top.zzf4.blog.service.AdminService;
import top.zzf4.blog.service.ArticleService;

import java.sql.SQLIntegrityConstraintViolationException;
import java.util.ArrayList;
import java.util.List;

@Service
public class AdminServiceImpl implements AdminService {

    @Autowired
    private ArticleService articleService;
    @Autowired
    private ArticleMapper articleMapper;
    @Autowired
    private TagsMapper tagMapper;

    @Override
    public void saveArticle(ArticleInsertDTO articleInsertDTO) {
        articleService.saveArticle(articleInsertDTO);
    }

    @Override
    public void updateArticle(ArticleUpdateDTO article) {
        articleService.updateArticle(article);
    }

    @Override
    public void deleteArticle(Long id) {
        articleService.deleteArticle(id);
    }

    @Override
    public void saveTag(TagInsertDTO tagInsertDTO) throws SQLIntegrityConstraintViolationException {
        articleService.saveTag(tagInsertDTO);
    }

    @Override
    public List<Tag> getTags() {
        return articleService.getTags();
    }

    @Override
    public List<Category> getArticleCategories() {
        return articleService.getArticleCategories();
    }

    @Override
    public void saveCategory(Category category) throws SQLIntegrityConstraintViolationException {
        articleService.saveCategory(category);
    }

    @Override
    public void deleteCategory(String id) {
        articleService.deleteCategory(id);
    }

    @Override
    public void updateCategory(Category category) {
        articleService.updateCategory(category);
    }

    @Override
    public void updateTag(Tag tag) {
        articleService.updateTag(tag);
    }

    @Override
    public void deleteTag(String id) {
        articleService.deleteTag(id);
    }

    @Override
    public PageResult<Articles> getArticleList(int pageSize, int currentPage) {
        PageResult<Articles> result = new PageResult<>();

        // 获取所有文章的基本信息
        List<Articles> articles = new ArrayList<>(articleMapper.selectList(new QueryWrapper<>()));

        // 获取文章的所有标签
        for (Articles article: articles) {
            article.setTags(tagMapper.getTagsByArticleId(article.getId()));
        }

        result.setList(articles);
        result.setCurr((long) currentPage);
        result.setTotal((long) articles.size());
        return result;
    }
}
