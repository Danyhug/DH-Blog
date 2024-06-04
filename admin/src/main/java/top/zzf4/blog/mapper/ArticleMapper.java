package top.zzf4.blog.mapper;

import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Select;
import top.zzf4.blog.entity.model.Article;

import java.util.List;

@Mapper
public interface ArticleMapper {
    @Insert("INSERT INTO Articles (title,content,category_id,publish_date,update_date,views,word_num) VALUES (#{title},#{content},#{categoryId},#{publishDate},#{updateDate},#{views},#{wordNum})")
    void saveArticle(Article article);

    void updateArticle(Article article);

    // @Select("SELECT id, title, content, publish_date, views FROM Articles")
    List<Article> getArticles(Integer categoryId);
}
