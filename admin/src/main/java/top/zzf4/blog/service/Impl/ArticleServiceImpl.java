package top.zzf4.blog.service.Impl;

import cn.hutool.core.bean.BeanUtil;
import cn.hutool.core.date.DateUtil;
import cn.hutool.json.JSONUtil;
import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import com.baomidou.mybatisplus.extension.service.impl.ServiceImpl;
import lombok.extern.log4j.Log4j2;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.scheduling.annotation.Async;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import top.zzf4.blog.constant.MessageConstant;
import top.zzf4.blog.constant.RedisConstant;
import top.zzf4.blog.entity.dto.ArticleInsertDTO;
import top.zzf4.blog.entity.dto.ArticleUpdateDTO;
import top.zzf4.blog.entity.dto.TagInsertDTO;
import top.zzf4.blog.entity.model.Articles;
import top.zzf4.blog.entity.model.Category;
import top.zzf4.blog.entity.model.CategoryDefaultTags;
import top.zzf4.blog.entity.model.Tag;
import top.zzf4.blog.entity.vo.OverviewCount;
import top.zzf4.blog.entity.vo.PageResult;
import top.zzf4.blog.mapper.ArticleMapper;
import top.zzf4.blog.mapper.CategoriesMapper;
import top.zzf4.blog.mapper.CategoryDefaultTagsMapper;
import top.zzf4.blog.mapper.TagsMapper;
import top.zzf4.blog.service.ArticleService;
import top.zzf4.blog.utils.RedisCacheUtils;

import java.sql.SQLIntegrityConstraintViolationException;
import java.util.*;
import java.util.concurrent.TimeUnit;
import java.util.stream.Collectors;

@Log4j2
@Service
public class ArticleServiceImpl extends ServiceImpl<ArticleMapper, Articles> implements ArticleService {

    @Autowired
    private ArticleMapper articleMapper;
    @Autowired
    private TagsMapper tagMapper;
    @Autowired
    private CategoriesMapper categoriesMapper;
    @Autowired
    private CategoryDefaultTagsMapper categoryDefaultTagsMapper;

    @Autowired
    private StringRedisTemplate stringRedisTemplate;

    @Autowired
    private RedisCacheUtils redisCacheUtils;
    @Autowired
    private QiniuServiceImpl qiniuServiceImpl;

    /**
     * 使用id查询文章信息
     *
     * @param id 文章id
     * @return 文章信息
     */
    @Override
    public Articles getArticleById(Long id) {
        // 1. 判断数据库中是否有缓存
        if (redisCacheUtils.hasNullKey(RedisConstant.CACHE_ARTICLE_ID + id)) {
            // 1.1 无缓存，新增缓存

            // 查询文章的信息
            Articles articles = this.getById(id);

            // 1.1.1 获取文章的分类id
            Integer categoryId = articles.getCategoryId();

            // 1.1.2 根据分类id查询默认标签的id
            Set<Long> tagsIdsSet = categoryDefaultTagsMapper.selectList(new LambdaQueryWrapper<CategoryDefaultTags>()
                    .eq(CategoryDefaultTags::getCategoryId, categoryId)).stream().map(CategoryDefaultTags::getTagId)
                    .collect(Collectors.toSet());

            // 1.1.3 遍历所有标签id查询出对应id信息
            List<Tag> tags = new ArrayList<>();
            if (!tagsIdsSet.isEmpty()) {
                tags = tagMapper.selectList(new LambdaQueryWrapper<Tag>().in(Tag::getId, tagsIdsSet));
            }

            // 1.1.4 再查询文章的标签信息
            List<Tag> tagsByArticleId = tagMapper.selectList(new LambdaQueryWrapper<Tag>().eq(Tag::getId, articles.getId()));

            // 1.1.5 tags 和 tagsByArticleId 做并集并去重
            Set<Tag> set = new HashSet<>();
            set.addAll(tags);
            set.addAll(tagsByArticleId);

            // 保存到文章信息中
            articles.setTags(new ArrayList<>(set));

            // 1.2 保存到 redis
            redisCacheUtils.setHash(RedisConstant.CACHE_ARTICLE_ID + id, BeanUtil.beanToMap(articles));
            System.out.println("已缓存文章信息");
        }

        // 2. 获取缓存数据
        Map<Object, Object> hash = redisCacheUtils.getHash(RedisConstant.CACHE_ARTICLE_ID + id);
        Articles articles = BeanUtil.toBean(hash, Articles.class);

        // 2.1 本次观看数据+1
        articles.setViews(articles.getViews() + 1);

        // 2.2 更新 redis 缓存观看数
        redisCacheUtils.updateHash(RedisConstant.CACHE_ARTICLE_ID + id, "views", articles.getViews());
        this.update().eq("id", id).set("views", articles.getViews()).update();

        // 返回数据
        return articles;
    }

    @Override
    public String getArticleTitleById(Long id) {
        return this.getOne(new LambdaQueryWrapper<Articles>().select(Articles::getTitle).eq(Articles::getId, id)).getTitle();
    }

    /**
     * 保存文章
     *
     */
    @Override
    public void saveArticle(ArticleInsertDTO articleInsertDTO) {
        Articles articles = new Articles();
        BeanUtils.copyProperties(articleInsertDTO, articles, "id");
        log.info("保存文章{}", articles);
        // 设置观看数
        articles.setViews(0);
        if (articleInsertDTO.getThumbnailUrl().isEmpty()) {
            // 没上传图片，就获取一个随机图片
            articles.setThumbnailUrl(qiniuServiceImpl.getRandomDefaultImage());
        }
        this.save(articles);

        // 查询标签slug对应id
        for (String tag : articleInsertDTO.getTags()) {
            // 临时标签数据
            // Tag tagTemp = tagMapper.selectBySlug(tag);
            Tag tagTemp = tagMapper.selectOne(new LambdaQueryWrapper<Tag>().eq(Tag::getSlug, tag));

            // 插入进postTags表中
            tagMapper.savePostTags(articles.getId(), tagTemp.getId());
        }

        // 删除首页缩略缓存
        redisCacheUtils.delete(RedisConstant.CACHE_ARTICLE_THUMBNAILS);
    }

    /**
     * 更新文章
     *
     */
    @Override
    public void updateArticle(ArticleUpdateDTO articleUpdateDTO) {
        Articles articles = new Articles();
        BeanUtils.copyProperties(articleUpdateDTO, articles);
        // 删除中间表的所有信息
        tagMapper.deleteByPostId(articles.getId());
        // 将标签插入
        List<String> tags = articleUpdateDTO.getTags();
        for (String tag : tags) {
            // 临时标签数据
            Tag tagTemp = tagMapper.selectOne(new LambdaQueryWrapper<Tag>().eq(Tag::getSlug, tag));

            // 插入进postTags表中
            tagMapper.savePostTags(articles.getId(), tagTemp.getId());
        }

        // 删除对应文章id
        redisCacheUtils.delete(RedisConstant.CACHE_ARTICLE_ID + articles.getId());
        // 删除首页缩略缓存
        redisCacheUtils.delete(RedisConstant.CACHE_ARTICLE_THUMBNAILS);

        if (articles.getThumbnailUrl().isEmpty()) {
            // 没上传图片，就获取一个随机图片
            articles.setThumbnailUrl(qiniuServiceImpl.getRandomDefaultImage());
        }
        this.updateById(articles);
    }

    /**
     * 删除文章
     *
     * @param id 文章id
     */
    @Override
    public void deleteArticle(Long id) {

    }

    /**
     * 保存标签
     *
     * @param tagInsertDTO 标签数据
     */
    @Override
    @Transactional
    public void saveTag(TagInsertDTO tagInsertDTO) throws SQLIntegrityConstraintViolationException {
        // 查看表中是否有相关值
        if (tagMapper.selectOne(new LambdaQueryWrapper<Tag>().eq(Tag::getSlug, tagInsertDTO.getSlug())) != null) {
            // 抛出主键异常
            throw new SQLIntegrityConstraintViolationException(MessageConstant.TAG_EXIST);
        }

        Tag tag = new Tag();
        tag.setName(tagInsertDTO.getName());
        tag.setSlug(tagInsertDTO.getSlug());

        tagMapper.insert(tag);
    }

    /**
     * 查询所有标签
     *
     * @return 标签列表
     */
    @Override
    public List<Tag> getTags() {
        return tagMapper.selectList(new QueryWrapper<>());
    }

    /**
     * 查询文章分类
     *
     * @return 分类列表
     */
    @Override
    public List<Category> getArticleCategories() {
        return articleMapper.getArticleCategories();
    }

    /**
     * 保存分类
     * @param category 分类
     */
    @Override
    public void saveCategory(Category category) throws SQLIntegrityConstraintViolationException {
        // 查看表中是否有相关值
        if (categoriesMapper.selectOne(new QueryWrapper<>(category).eq("slug", category.getSlug())) != null) {
            throw new SQLIntegrityConstraintViolationException(MessageConstant.CATEGORY_EXIST);
        }

        categoriesMapper.insert(category);
    }

    /**
     * 删除分类
     * @param id 文章id
     */
    @Override
    public void deleteCategory(String id) {
        categoriesMapper.deleteById(id);
    }

    /**
     * 根据id查询分类
     * @param id 文章aid
     * @return 分类
     */
    @Override
    public Category getCategoryById(String id) {
        return categoriesMapper.selectById(id);
    }

    /**
     * 更新分类
     *
     * @param category 分类
     */
    @Override
    public void updateCategory(Category category) {
        categoriesMapper.updateById(category);
    }

    @Override
    public void saveCategoryDefaultTags(Long categoryId, Long[] tagIds) {
        // 能执行到这里说明 categoryId 肯定存在

        // 删除之前的数据
        categoryDefaultTagsMapper.delete(new LambdaQueryWrapper<CategoryDefaultTags>().eq(CategoryDefaultTags::getCategoryId, categoryId));

        // 遍历tags
        for (Long tagId: tagIds) {
            // 插入新的数据
            categoryDefaultTagsMapper.insert(
                    CategoryDefaultTags.builder()
                            .categoryId(categoryId)
                            .tagId(tagId)
                            .build()
            );
        }
    }

    /**
     * 更新标签
     *
     * @param tag 标签
     */
    @Override
    public void updateTag(Tag tag) {
        tagMapper.updateById(tag);
    }

    /**
     * 删除标签
     *
     * @param id 标签id
     */
    @Override
    public void deleteTag(String id) {
        tagMapper.deleteById(id);
    }

    @Override
    public List<Long> getCategoryDefaultTagsById(Long id) {
        return categoryDefaultTagsMapper.selectList(new LambdaQueryWrapper<CategoryDefaultTags>().eq(CategoryDefaultTags::getCategoryId, id)).stream().map(CategoryDefaultTags::getTagId).collect(Collectors.toList());
    }

    @Override
    public OverviewCount getOverview() {
        if (redisCacheUtils.hasNullKey(RedisConstant.CACHE_OVERVIEW)) {
            // 创建 key
            Long tagCount = tagMapper.selectCount(new LambdaQueryWrapper<Tag>().select(Tag::getId));
            Long articleCount = articleMapper.selectCount(new LambdaQueryWrapper<Articles>().select(Articles::getId));
            Long commentCount = 4396L;

            OverviewCount overviewCount = OverviewCount.builder()
                    .articleCount(articleCount)
                    .categoryCount(categoriesMapper.selectCount(new LambdaQueryWrapper<Category>().select(Category::getId)))
                    .tagCount(tagCount)
                    .commentCount(commentCount)
                    .build();

            // 生成缓存
            redisCacheUtils.set(RedisConstant.CACHE_OVERVIEW, JSONUtil.toJsonStr(overviewCount));
            redisCacheUtils.setExpire(RedisConstant.CACHE_OVERVIEW, 12, TimeUnit.HOURS);
            return overviewCount;
        }
        return JSONUtil.toBean( (String) redisCacheUtils.get(RedisConstant.CACHE_OVERVIEW), OverviewCount.class);
    }

    @Async
    @Override
    public void pv() {
        // 计算今日的key值
        String key = RedisConstant.CACHE_DAILY_PV + DateUtil.format(new Date(), "yyyy-MM-dd");
        if (redisCacheUtils.hasNullKey(key)) {
            // 不存在key，新建+1
            redisCacheUtils.set(key, 1L, 48L, TimeUnit.HOURS);
            return;
        }

        redisCacheUtils.incr(key);
    }

    /**
     * 分页查询缓存首页的文章缩略信息
     * 从redis中返回 不带内容 的文章基本信息列表，文章按照id倒序排列
     * @return 文章缩略信息列表
     */
    @Override
    public PageResult<Articles> getArticleThumbnail(int pageSize, int currentPage) {
        PageResult<Articles> result = new PageResult<>();

        // 1. 若本地无缓存
        if (redisCacheUtils.hasNullKey(RedisConstant.CACHE_ARTICLE_THUMBNAILS)) {
            // 1.1 查询数据库数据
            // 获取所有文章的基本信息
            List<Articles> articles = new ArrayList<>(articleMapper.selectList(new LambdaQueryWrapper<Articles>()
                    .select(Articles::getId, Articles::getTitle, Articles::getThumbnailUrl, Articles::getCreateTime,
                            Articles::getViews, Articles::getWordNum, Articles::getIsLocked)));

            // 所有分数
            ArrayList<Double> scores = new ArrayList<>();
            // 1.2 获取文章的所有标签
            for (Articles article: articles) {
                article.setTags(tagMapper.getTagsByArticleId(article.getId()));
                scores.add(Double.valueOf(article.getId()));
            }

            // 1.3 更新缓存
            // 批量插入缓存
            redisCacheUtils.batchSetZSet(RedisConstant.CACHE_ARTICLE_THUMBNAILS, articles, scores);
            log.info("已缓存首页缩略文章信息");
        }

        // 有缓存的情况下，获取缓存
        long card = redisCacheUtils.getZSetCard(RedisConstant.CACHE_ARTICLE_THUMBNAILS);

        // 总页数
        long totalPage = (long) Math.ceil((double) card / pageSize);
        result.setTotal(totalPage);

        result.setCurr((long) currentPage);
        result.setList(stringRedisTemplate.opsForZSet().reverseRange(RedisConstant.CACHE_ARTICLE_THUMBNAILS, (long) (currentPage - 1) * pageSize, (long) currentPage * pageSize - 1)
                .stream().map(s -> JSONUtil.toBean(s, Articles.class)).collect(Collectors.toList()));

        return result;
    }
}
