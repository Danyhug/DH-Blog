package top.zzf4.blog.mapper;

import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Select;
import top.zzf4.blog.entity.model.Article;
import top.zzf4.blog.entity.model.Tag;

import java.util.List;

@Mapper
public interface TagsMapper {
    @Insert("INSERT INTO Tags (name,slug,created_at,updated_at) VALUES (#{name},#{slug},#{createdAt},#{updatedAt})")
    void saveTag(Tag tag);

    @Select("SELECT * FROM Tags")
    List<Tag> getTags();

    @Select("SELECT * FROM Articles WHERE id = #{id}")
    Article selectById(Long id);

    // 根据slug查询标签
    @Select("SELECT * FROM Tags WHERE slug = #{slug}")
    Tag selectBySlug(String slug);

    // 将标签插入posttags表中
    @Insert("INSERT INTO PostTags (post_id,tag_id) VALUES (#{postId},#{tagId})")
    void savePostTags(Long postId, Long tagId);
}
