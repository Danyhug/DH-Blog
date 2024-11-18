package top.zzf4.blog.service.Impl;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.extension.service.impl.ServiceImpl;
import org.springframework.stereotype.Service;
import top.zzf4.blog.entity.model.Comment;
import top.zzf4.blog.entity.vo.PageResult;
import top.zzf4.blog.mapper.CommentMapper;
import top.zzf4.blog.service.CommentService;

import java.util.List;

@Service
public class CommentServiceImpl extends ServiceImpl<CommentMapper, Comment> implements CommentService {
    @Override
    public boolean addComment(Comment comment) {
        return this.save(comment);
    }

    @Override
    public PageResult<Comment> getCommentList(Long articleId, int pageSize, int pageNum) {
        List<Comment> list = this.list(new LambdaQueryWrapper<Comment>()
                .eq(Comment::getArticleId, articleId).eq(Comment::getIsPublic, true)).stream().toList();

        // 如果parentId为null，则为一级元素，如果不为null，则添加到对应的id下
        List<Comment> result = list.stream().filter(comment -> comment.getParentId() == null).toList();
        result.forEach(comment -> comment.setChildren(list.stream().filter(c -> c.getParentId() != null && c.getParentId().equals(comment.getId())).toList()));
        return new PageResult<>((long) result.size(), (long) pageNum, result);
    }
}
