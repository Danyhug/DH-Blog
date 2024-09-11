package top.zzf4.blog.utils;

import cn.hutool.json.JSONUtil;
import lombok.extern.log4j.Log4j2;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.data.redis.core.RedisCallback;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Component;
import top.zzf4.blog.constant.RedisConstant;
import top.zzf4.blog.entity.model.Articles;

import java.util.List;
import java.util.concurrent.TimeUnit;

@Log4j2
@Component
public class RedisCacheUtils {

    @Autowired
    private StringRedisTemplate redisTemplate;

    // 设置普通缓存
    public void set(String key, Object value) {
        redisTemplate.opsForValue().set(key, (String) value);
    }

    public void set(String key, Object value, Long time, TimeUnit unit) {
        redisTemplate.opsForValue().set(key, (String) value, time, unit);
    }

    // 获取普通缓存
    public Object get(String key) {
        return redisTemplate.opsForValue().get(key);
    }

    // 设置 zSet
    public void setZSet(String key, Object value, Double score) {
        redisTemplate.opsForZSet().add(key, (String) value, score);
    }

    // 批量设置 zSet
    public <T> void batchSetZSet(String key, List<T> values, List<Double> scores) {
        if (values.size() != scores.size()) {
            log.error("批量设置 zSet 的参数长度不一致");
            throw new RuntimeException("批量设置 zSet 的参数长度不一致");
        }

        redisTemplate.executePipelined((RedisCallback<Object>) connection -> {
            for (int i = 0; i < values.size(); i++) {
                connection.zAdd(key.getBytes(), scores.get(i), JSONUtil.toJsonStr(values.get(i)).getBytes());
            }
            return null;
        });
    }

    // 获取 zSet 的 card
    public long getZSetCard(String key) {
        Long card = redisTemplate.opsForZSet().zCard(key);
        if (card == null) {
            log.error("获取 zSet 的 card 失败 {}", key);
            throw new RuntimeException("获取 zSet 的 card 失败" + key);
        }

        return card;
    }

    // 判断 key 是否存在
    public boolean hasKey(String key) {
        return Boolean.TRUE.equals(redisTemplate.hasKey(key));
    }
}
