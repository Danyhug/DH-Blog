package top.zzf4.blog.service.Impl;

import ch.qos.logback.core.testUtil.RandomUtil;
import com.github.pagehelper.PageHelper;
import lombok.extern.log4j.Log4j2;
import org.apache.tomcat.util.http.fileupload.FileUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.core.io.ClassPathResource;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.multipart.MultipartFile;
import top.zzf4.blog.constant.MessageConstant;
import top.zzf4.blog.entity.dto.ArticleInsertDTO;
import top.zzf4.blog.entity.dto.ArticlePageDTO;
import top.zzf4.blog.entity.dto.ArticleUpdateDTO;
import top.zzf4.blog.entity.dto.TagInsertDTO;
import top.zzf4.blog.entity.model.Article;
import top.zzf4.blog.entity.model.Category;
import top.zzf4.blog.entity.model.Tag;
import top.zzf4.blog.entity.vo.PageResult;
import top.zzf4.blog.mapper.ArticleMapper;
import top.zzf4.blog.mapper.CategoriesMapper;
import top.zzf4.blog.mapper.TagsMapper;
import top.zzf4.blog.service.ArticleService;

import java.io.File;
import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.sql.SQLIntegrityConstraintViolationException;
import java.time.LocalDateTime;
import java.util.List;
import java.util.Random;
import java.util.UUID;

@Log4j2
@Service
public class ArticleServiceImpl implements ArticleService {
    @Autowired
    private ArticleMapper articleMapper;
    @Autowired
    private TagsMapper tagMapper;
    @Autowired
    private CategoriesMapper categoriesMapper;

    /**
     * 使用id查询文章信息
     *
     * @param id 文章id
     * @return 文章信息
     */
    @Override
    public Article getArticleById(Long id) {
        // 查询文章的信息
        Article article = articleMapper.selectById(id);
        // 再查询文章的标签信息
        List<Tag> tagsByArticleId = tagMapper.getTagsByArticleId(id);
        article.setTags(tagsByArticleId);
        return article;
    }

    /**
     * 保存文章
     *
     */
    @Override
    public void saveArticle(ArticleInsertDTO articleInsertDTO) {
        Article article = new Article();
        BeanUtils.copyProperties(articleInsertDTO, article);
        log.info("保存文章{}", article);
        // 设置观看数
        article.setViews(0);
        LocalDateTime date = LocalDateTime.now();
        article.setPublishDate(date);
        article.setUpdateDate(date);
        articleMapper.saveArticle(article);

        // 查询标签slug对应id
        for (String tag : articleInsertDTO.getTags()) {
            // 临时标签数据
            Tag tagTemp = tagMapper.selectBySlug(tag);

            // 插入进postTags表中
            tagMapper.savePostTags(article.getId(), tagTemp.getId());
        }
    }

    /**
     * 更新文章
     *
     */
    @Override
    public void updateArticle(ArticleUpdateDTO articleUpdateDTO) {
        Article article = new Article();
        BeanUtils.copyProperties(articleUpdateDTO, article);
        article.setUpdateDate(LocalDateTime.now());
        // 删除中间表的所有信息
        tagMapper.deleteByPostId(article.getId());
        // 将标签插入
        List<String> tags = articleUpdateDTO.getTags();
        for (String tag : tags) {
            // 临时标签数据
            Tag tagTemp = tagMapper.selectBySlug(tag);

            // 插入进postTags表中
            tagMapper.savePostTags(article.getId(), tagTemp.getId());
        }

        log.info("更新的article属性为 {}", article);
        articleMapper.updateArticle(article);
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
        if (tagMapper.selectBySlug(tagInsertDTO.getSlug()) != null) {
            // 抛出主键异常
            throw new SQLIntegrityConstraintViolationException(MessageConstant.TAG_EXIST);
        }

        Tag tag = new Tag();
        LocalDateTime date = LocalDateTime.now();

        tag.setName(tagInsertDTO.getName());
        tag.setSlug(tagInsertDTO.getSlug());
        tag.setCreatedAt(date);
        tag.setUpdatedAt(date);
        tagMapper.saveTag(tag);
    }

    /**
     * 查询所有标签
     *
     * @return 标签列表
     */
    @Override
    public List<Tag> getTags() {
        return tagMapper.getTags();
    }

    /**
     * 分页查询文章
     */
    @Override
    public PageResult<Article> getPage(ArticlePageDTO articlePage) {
        PageHelper.startPage(articlePage.getPageNum(), articlePage.getPageSize());
        List<Article> articles = articleMapper.getArticles(articlePage.getCategoryId());
        for (Article article : articles) {
            article.setTags(tagMapper.getTagsByArticleId(article.getId()));
        }
        return new PageResult<>(articles.size(), articles);
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
        if (categoriesMapper.selectBySlug(category.getSlug()) != null) {
            throw new SQLIntegrityConstraintViolationException(MessageConstant.CATEGORY_EXIST);
        }

        LocalDateTime date = LocalDateTime.now();
        category.setCreatedAt(date);
        category.setUpdatedAt(date);
        categoriesMapper.saveCategory(category);
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
        category.setUpdatedAt(LocalDateTime.now());
        categoriesMapper.updateCategory(category);
    }

    /**
     * 更新标签
     *
     * @param tag 标签
     */
    @Override
    public void updateTag(Tag tag) {
        tag.setUpdatedAt(LocalDateTime.now());
        tagMapper.updateTag(tag);
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

    /**
     * 返回随机图片
     *
     * @return 图片字节
     */
    @Override
    public byte[] getRandomImage() throws IOException {
        int num = new Random().nextInt(5) + 1;

        // 从资源文件加载图片（这里假设图片位于 classpath:/static 目录下）
        ClassPathResource resource = new ClassPathResource("static/articleBg/" + num + ".jpg");

        // 返回图片字节数组
        return Files.readAllBytes(resource.getFile().toPath());
    }
}
