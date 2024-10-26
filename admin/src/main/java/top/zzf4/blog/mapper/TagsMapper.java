package top.zzf4.blog.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import org.apache.ibatis.annotations.Delete;
import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Select;
import top.zzf4.blog.entity.model.Tag;

import java.util.List;

@Mapper
public interface TagsMapper extends BaseMapper<Tag> {
    // 通过文章id查询文章的所属标签信息
    @Select("SELECT * FROM Tags t INNER JOIN PostTags pt ON t.id = pt.tag_id WHERE pt.post_id = #{id}")
    List<Tag> getTagsByArticleId(Long id);

    // 根据post_id删除所有记录
    @Delete("DELETE FROM PostTags WHERE post_id = #{postId}")
    void deleteByPostId(Long postId);

    // 将标签插入posttags表中
    @Insert("INSERT INTO PostTags (post_id,tag_id) VALUES (#{postId},#{tagId})")
    void savePostTags(Long postId, Long tagId);
}
