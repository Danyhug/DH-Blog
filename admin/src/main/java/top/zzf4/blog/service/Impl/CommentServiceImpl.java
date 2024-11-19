package top.zzf4.blog.service.Impl;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.extension.service.impl.ServiceImpl;
import org.springframework.stereotype.Service;
import top.zzf4.blog.entity.model.Comment;
import top.zzf4.blog.entity.vo.PageResult;
import top.zzf4.blog.mapper.CommentMapper;
import top.zzf4.blog.service.CommentService;

import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

@Service
public class CommentServiceImpl extends ServiceImpl<CommentMapper, Comment> implements CommentService {
    @Override
    public boolean addComment(Comment comment) {
        return this.save(comment);
    }

    @Override
    public PageResult<Comment> getCommentListByArticle(Long articleId, int pageSize, int pageNum) {
        // 查询评论列表
        List<Comment> list = this.list(new LambdaQueryWrapper<Comment>()
                .eq(Comment::getArticleId, articleId)
                .eq(Comment::getIsPublic, true));

        // 构建评论树
        Map<Integer, List<Comment>> childrenMap = list.stream()
                .filter(comment -> comment.getParentId() != null)
                .collect(Collectors.groupingBy(Comment::getParentId));

        List<Comment> result = list.stream()
                .filter(comment -> comment.getParentId() == null)
                .peek(comment -> setChildrenRecursively(comment, childrenMap))
                .collect(Collectors.toList());

        // 返回分页结果
        return new PageResult<>((long) list.size(), (long) pageNum, result);
    }

    @Override
    public PageResult<Comment> getCommentList(int pageSize, int pageNum) {
        // 查询所有评论
        List<Comment> list = this.list();
        return new PageResult<>((long) list.size(), (long) pageNum, list);
    }

    private void setChildrenRecursively(Comment parentComment, Map<Integer, List<Comment>> childrenMap) {
        List<Comment> children = childrenMap.getOrDefault(parentComment.getId(), List.of());
        parentComment.setChildren(children);
        children.forEach(child -> setChildrenRecursively(child, childrenMap));
    }
}
