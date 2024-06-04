package top.zzf4.blog.service.Impl;

import com.github.pagehelper.PageHelper;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import top.zzf4.blog.entity.dto.ArticleInsertDto;
import top.zzf4.blog.entity.dto.ArticlePageDTO;
import top.zzf4.blog.entity.dto.TagInsertDTO;
import top.zzf4.blog.entity.model.Article;
import top.zzf4.blog.entity.model.Tag;
import top.zzf4.blog.entity.vo.PageResult;
import top.zzf4.blog.mapper.ArticleMapper;
import top.zzf4.blog.mapper.TagsMapper;
import top.zzf4.blog.service.ArticleService;

import java.time.LocalDate;
import java.time.LocalDateTime;
import java.util.List;

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
        return tagMapper.selectById(id);
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
        article.setCategoryId(1);

        // 设置观看数
        article.setViews(0);
        LocalDateTime date = LocalDateTime.now();
        article.setPublishDate(date);
        article.setUpdateDate(date);
        articleMapper.saveArticle(article);
    }

    /**
     * 更新文章
     *
     * @param article
     */
    @Override
    public void updateArticle(Article article) {
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
        return new PageResult<>(articles.size(), articles);
    }
}
