package top.zzf4.blog.service.Impl;

import com.github.pagehelper.PageHelper;
import lombok.extern.log4j.Log4j2;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import top.zzf4.blog.entity.dto.ArticleInsertDto;
import top.zzf4.blog.entity.dto.ArticlePageDTO;
import top.zzf4.blog.entity.dto.ArticleUpdateDTO;
import top.zzf4.blog.entity.dto.TagInsertDTO;
import top.zzf4.blog.entity.model.Article;
import top.zzf4.blog.entity.model.Category;
import top.zzf4.blog.entity.model.Tag;
import top.zzf4.blog.entity.vo.PageResult;
import top.zzf4.blog.mapper.ArticleMapper;
import top.zzf4.blog.mapper.TagsMapper;
import top.zzf4.blog.service.ArticleService;

import java.time.LocalDate;
import java.time.LocalDateTime;
import java.util.List;

@Log4j2
@Service
public class ArticleServiceImpl implements ArticleService {
    @Autowired
    private ArticleMapper articleMapper;
    @Autowired
    private TagsMapper tagMapper;

    /**
     * 使用id查询文章信息
     *
     * @param id
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
    public void saveArticle(ArticleInsertDto articleInsertDto) {
        Article article = new Article();
        article.setTitle(articleInsertDto.getTitle());
        article.setContent(articleInsertDto.getContent());
        article.setCategoryId(articleInsertDto.getCategoryId());
        // 设置观看数
        article.setViews(0);
        LocalDateTime date = LocalDateTime.now();
        article.setPublishDate(date);
        article.setUpdateDate(date);
        articleMapper.saveArticle(article);

        // 查询标签slug对应id
        for (String tag : articleInsertDto.getTags()) {
            // 临时标签数据
            Tag tagTemp = tagMapper.selectBySlug(tag);

            // 插入进postTags表中
            tagMapper.savePostTags(article.getId(), tagTemp.getId());
        }
    }

    /**
     * 更新文章
     *
     * @param articleUpdateDTO
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
     * @param id
     */
    @Override
    public void deleteArticle(Long id) {

    }

    /**
     * 保存标签
     *
     * @param tagInsertDTO
     */
    @Override
    public void saveTag(TagInsertDTO tagInsertDTO) {
        Tag tag = new Tag();
        LocalDateTime date = LocalDateTime.now();

        tag.setName(tagInsertDTO.getName());
        tag.setSlug(tagInsertDTO.getSlug());
        tag.setCreatedAt(date);
        tag.setUpdatedAt(date);
        System.out.println(tag);
        tagMapper.saveTag(tag);
    }

    /**
     * 查询所有标签
     *
     * @return
     */
    @Override
    public List<Tag> getTags() {
        return tagMapper.getTags();
    }

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
     * @return
     */
    @Override
    public List<Category> getArticleCategories() {
        return articleMapper.getArticleCategories();
    }
}
