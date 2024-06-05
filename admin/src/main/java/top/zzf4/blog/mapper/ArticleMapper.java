package top.zzf4.blog.mapper;

import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Select;
import top.zzf4.blog.entity.model.Article;
import top.zzf4.blog.entity.model.Category;

import java.util.List;

@Mapper
public interface ArticleMapper {
    void saveArticle(Article article);

    void updateArticle(Article article);

    List<Article> getArticles(Integer categoryId);

    @Select("SELECT * FROM Articles WHERE id = #{id}")
    Article selectById(Long id);

    @Select("SELECT * FROM Categories")
    List<Category> getArticleCategories();
}
