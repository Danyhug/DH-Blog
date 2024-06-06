package top.zzf4.blog.mapper;

import org.apache.ibatis.annotations.*;
import top.zzf4.blog.entity.model.Category;

@Mapper
public interface CategoriesMapper {
    @Insert("INSERT INTO Categories (name,slug,created_at,updated_at) VALUES (#{name},#{slug},#{createdAt},#{updatedAt})")
    void saveCategory(Category category);

    // 使用slug查询信息
    @Select("SELECT * FROM Categories WHERE id = #{id}")
    Category selectById(String id);

    @Update("UPDATE Categories SET name = #{name},slug = #{slug},updated_at = #{updatedAt} WHERE id = #{id}")
    void updateCategory(Category category);

    @Select("SELECT * FROM Categories WHERE slug = #{slug}")
    Category selectBySlug(String slug);

    @Delete("DELETE FROM Categories WHERE id = #{id}")
    void deleteById(String id);
}
