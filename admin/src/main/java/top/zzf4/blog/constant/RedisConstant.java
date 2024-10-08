package top.zzf4.blog.constant;

// 定义redis常量
public class RedisConstant {
    // 缓存文章信息
    public static final String CACHE_ARTICLE_ID = "dhBlog:cache:article:";

    // 首页缓存文章列表
    public static final String CACHE_ARTICLE_THUMBNAILS = "dhBlog:cache:article:thumbnails";

    // 七牛云缓存的默认展示图片
    public static final String CACHE_QINIU_DEFAULT_IMAGE = "dhBlog:cache:qiniu:defaultImage";
}
