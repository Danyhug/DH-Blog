package top.zzf4.blog.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import org.apache.ibatis.annotations.Mapper;
import top.zzf4.blog.entity.model.AccessLog;

@Mapper
public interface AccessLogMapper extends BaseMapper<AccessLog> {
}