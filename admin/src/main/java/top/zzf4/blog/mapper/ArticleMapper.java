package top.zzf4.blog.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import org.apache.ibatis.annotations.Select;
import top.zzf4.blog.entity.model.Articles;
import top.zzf4.blog.entity.model.Category;

import java.util.List;

public interface ArticleMapper extends BaseMapper<Articles> {
    List<Articles> getArticles(Integer categoryId);

    @Select("SELECT * FROM Categories")
    List<Category> getArticleCategories();
}
