package top.zzf4.blog.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import org.apache.ibatis.annotations.*;
import top.zzf4.blog.entity.model.Tag;

import java.util.List;

@Mapper
public interface TagsMapper extends BaseMapper<Tag> {
    @Insert("INSERT INTO Tags (name,slug,created_at,updated_at) VALUES (#{name},#{slug},#{createdAt},#{updatedAt})")
    void saveTag(Tag tag);

    @Select("SELECT * FROM Tags")
    List<Tag> getTags();

    // 通过文章id查询文章的所属标签信息
    @Select("SELECT * FROM Tags t INNER JOIN PostTags pt ON t.id = pt.tag_id WHERE pt.post_id = #{id}")
    List<Tag> getTagsByArticleId(Long id);

    // 根据slug查询标签
    @Select("SELECT * FROM Tags WHERE slug = #{slug}")
    Tag selectBySlug(String slug);

    // 根据post_id删除所有记录
    @Delete("DELETE FROM PostTags WHERE post_id = #{postId}")
    void deleteByPostId(Long postId);

    // 更新标签记录
    @Update("UPDATE Tags SET name = #{name},slug = #{slug},updated_at = #{updatedAt} WHERE id = #{id}")
    void updateTag(Tag tag);

    // 将标签插入posttags表中
    @Insert("INSERT INTO PostTags (post_id,tag_id) VALUES (#{postId},#{tagId})")
    void savePostTags(Long postId, Long tagId);

    // 删除标签
    @Delete("DELETE FROM Tags WHERE id = #{id}")
    void deleteById(String id);
}
