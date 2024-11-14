package top.zzf4.blog.config;

import com.baomidou.mybatisplus.core.handlers.MetaObjectHandler;
import org.apache.ibatis.reflection.MetaObject;
import org.springframework.context.annotation.Configuration;

import java.time.LocalDateTime;
import java.time.ZoneId;

@Configuration
public class MybatisObjectHandler implements MetaObjectHandler {

    private LocalDateTime getLocalTime() {
        ZoneId zoneId = ZoneId.of("Asia/Shanghai");
        return LocalDateTime.now(zoneId);
    }

    @Override
    public void insertFill(MetaObject metaObject) {
        setFieldValByName("createTime", this.getLocalTime(), metaObject);
        setFieldValByName("updateTime", this.getLocalTime(), metaObject);
        setFieldValByName("accessTime", this.getLocalTime(), metaObject);
    }

    @Override
    public void updateFill(MetaObject metaObject) {
        setFieldValByName("updateTime", this.getLocalTime(), metaObject);
    }
}