package top.zzf4.blog.service.Impl;

import com.qiniu.storage.BucketManager;
import com.qiniu.storage.Configuration;
import com.qiniu.storage.Region;
import com.qiniu.storage.model.FileInfo;
import com.qiniu.util.Auth;
import lombok.extern.log4j.Log4j2;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;
import top.zzf4.blog.config.QiniuProperties;
import top.zzf4.blog.constant.RedisConstant;
import top.zzf4.blog.utils.RedisCacheUtils;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.concurrent.TimeUnit;

@Log4j2
@Service
public class QiniuServiceImpl {
    @Autowired
    private QiniuProperties qiniuProperties;

    @Autowired
    private RedisCacheUtils redisCacheUtils;

    /**
     * 获取指定目录的文件列表
     * @param folderName 目录名称
     * @return 文件列表
     */
    public FileInfo[] getFileList(String folderName) {
        // 构造一个带指定 Region 对象的配置类
        Configuration cfg = new Configuration(Region.qvmHuabei());

        Auth auth = Auth.create(qiniuProperties.getAccessKey(), qiniuProperties.getSecretKey());
        BucketManager bucketManager = new BucketManager(auth, cfg);

    // 文件名前缀
    String prefix = "";
    // 每次迭代的长度限制，最大1000，推荐值 1000
    int limit = 1000;
    // 指定目录分隔符，列出所有公共前缀（模拟列出目录效果）。缺省值为空字符串
    String delimiter = "";

    //列举空间文件列表
    BucketManager.FileListIterator fileListIterator = bucketManager.createFileListIterator(
            qiniuProperties.getBucket(), prefix, limit, delimiter
    );

    // 返回的文件列表信息
    List<FileInfo> result = new ArrayList<>();
    while (fileListIterator.hasNext()) {
        //处理获取的file list结果
        FileInfo[] items = fileListIterator.next();
        for (FileInfo item : items) {
            // 判断不是目录且包含目录名的文件
            if (item.key.contains(folderName) && !item.key.equals(folderName + "/")) result.add(item);
        }
    }

    return result.toArray(new FileInfo[0]);
    }

    /**
     * Redis 中缓存预热，将需要的七牛云数据缓存到 Redis 中
     * 目前是缓存默认的展示图片，三天更新一次
     */
    @Scheduled(fixedDelay = 1000 * 60 * 60 * 24 * 3, initialDelay = 1000)
    public void initCache() {
        // 获取默认的图片列表名
        List<String> defaultArticleImages = Arrays.stream(getFileList(qiniuProperties.getDefaultImageName()))
                .map(item -> item.key).toList();
        // 缓存到 Redis 中
        if (!redisCacheUtils.hasNullKey(RedisConstant.CACHE_QINIU_DEFAULT_IMAGE)) {
            // 存在就删除
            redisCacheUtils.delete(RedisConstant.CACHE_QINIU_DEFAULT_IMAGE);
        }
        redisCacheUtils.setList(RedisConstant.CACHE_QINIU_DEFAULT_IMAGE, defaultArticleImages);
        redisCacheUtils.setExpire(RedisConstant.CACHE_QINIU_DEFAULT_IMAGE, 1, TimeUnit.DAYS);
        log.info("缓存默认展示图片成功");
    }

    public String getRandomDefaultImage() {
        try {
            return redisCacheUtils.getRandomListValue(RedisConstant.CACHE_QINIU_DEFAULT_IMAGE);
        } catch (RuntimeException e) {
            log.info(e.getMessage());
            this.initCache();
            return redisCacheUtils.getRandomListValue(RedisConstant.CACHE_QINIU_DEFAULT_IMAGE);
        }
    }
}
