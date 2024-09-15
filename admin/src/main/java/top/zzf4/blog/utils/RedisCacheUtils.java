package top.zzf4.blog.utils;

import cn.hutool.core.util.RandomUtil;
import cn.hutool.json.JSONUtil;
import lombok.extern.log4j.Log4j2;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.data.redis.core.RedisCallback;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Component;

import java.util.List;
import java.util.Map;
import java.util.concurrent.TimeUnit;

@Log4j2
@Component
public class RedisCacheUtils {

    @Autowired
    private StringRedisTemplate stringRedisTemplate;
    @Autowired
    private RedisTemplate<String, Object> redisTemplate;


    // 设置普通缓存
    public void set(String key, Object value) {
        stringRedisTemplate.opsForValue().set(key, (String) value);
    }

    public void set(String key, Object value, Long time, TimeUnit unit) {
        stringRedisTemplate.opsForValue().set(key, (String) value, time, unit);
    }

    // 获取普通缓存
    public Object get(String key) {
        return stringRedisTemplate.opsForValue().get(key);
    }

    // 设置列表
    public <T> void setList(String key, List<T> values) {
        stringRedisTemplate.opsForList().rightPushAll(key, values.stream().map(JSONUtil::toJsonStr).toList());
    }

    // 列表中随机获取值
    public String getRandomListValue(String key) {
        // 获取长度
        long size = stringRedisTemplate.opsForList().size(key);
        // 获取随机索引
        long index = RandomUtil.randomInt((int) size);
        return stringRedisTemplate.opsForList().index(key, index);
    }


    // 设置 zSet
    public void setZSet(String key, Object value, Double score) {
        stringRedisTemplate.opsForZSet().add(key, (String) value, score);
    }

    // 批量设置 zSet
    public <T> void batchSetZSet(String key, List<T> values, List<Double> scores) {
        if (values.size() != scores.size()) {
            log.error("批量设置 zSet 的参数长度不一致");
            throw new RuntimeException("批量设置 zSet 的参数长度不一致");
        }

        stringRedisTemplate.executePipelined((RedisCallback<Object>) connection -> {
            for (int i = 0; i < values.size(); i++) {
                connection.zAdd(key.getBytes(), scores.get(i), JSONUtil.toJsonStr(values.get(i)).getBytes());
            }
            return null;
        });
    }

    // 获取 zSet 的 card
    public long getZSetCard(String key) {
        Long card = stringRedisTemplate.opsForZSet().zCard(key);
        if (card == null) {
            log.error("获取 zSet 的 card 失败 {}", key);
            throw new RuntimeException("获取 zSet 的 card 失败" + key);
        }

        return card;
    }

    /**
     * 判断 key 是否为空，true 为不存在key
     * @param key
     * @return
     */
    public boolean hasNullKey(String key) {
        return Boolean.FALSE.equals(stringRedisTemplate.hasKey(key));
    }

    // 设置过期时间
    public void setExpire(String key, long time, TimeUnit unit) {
        stringRedisTemplate.expire(key, time, unit);
    }

    // 设置 hash 缓存（方便增加观看数）
    public <K, V> void setHash(String key, Map<K, V> hashMap) {
        redisTemplate.opsForHash().putAll(key, hashMap);
    }

    // 更新 hash 指定 key 的缓存
    public <K, V> void updateHash(String key, K hashKey, V value) {
        redisTemplate.opsForHash().put(key, hashKey, value);
    }

    // 获取 hash 缓存
    public Map<Object, Object> getHash(String key) {
        return redisTemplate.opsForHash().entries(key);
    }

    // 删除缓存
    public void delete(String key) {
        stringRedisTemplate.delete(key);
    }
}
