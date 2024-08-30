package top.zzf4.blog.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import org.apache.ibatis.annotations.*;
import top.zzf4.blog.entity.model.Category;

@Mapper
public interface CategoriesMapper extends BaseMapper<Category> {
}
