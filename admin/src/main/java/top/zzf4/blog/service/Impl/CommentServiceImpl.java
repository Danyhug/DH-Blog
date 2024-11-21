package top.zzf4.blog.service.Impl;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.extension.service.impl.ServiceImpl;
import org.springframework.scheduling.annotation.Async;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import top.zzf4.blog.entity.model.Comment;
import top.zzf4.blog.entity.vo.PageResult;
import top.zzf4.blog.mapper.CommentMapper;
import top.zzf4.blog.service.CommentService;

import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

@Service
public class CommentServiceImpl extends ServiceImpl<CommentMapper, Comment> implements CommentService {
    private final CommentMapper commentMapper;

    public CommentServiceImpl(CommentMapper commentMapper) {
        this.commentMapper = commentMapper;
    }

    @Override
    public boolean addComment(Comment comment) {
        return this.save(comment);
    }

    @Override
    public PageResult<Comment> getCommentListByArticle(Long articleId, int pageSize, int pageNum) {
        // 查询评论列表
        LambdaQueryWrapper<Comment> eq = new LambdaQueryWrapper<Comment>()
                .eq(Comment::getArticleId, articleId)
                .eq(Comment::getIsPublic, true);

        // 返回分页结果
        return getCommentList(pageSize, pageNum, eq);
    }

    @Override
    public PageResult<Comment> getCommentList(int pageSize, int pageNum, LambdaQueryWrapper<Comment> queryWrapper) {
        if (queryWrapper == null) queryWrapper = new LambdaQueryWrapper<>();
        // 查询所有评论
        List<Comment> list = this.list(queryWrapper.orderByDesc(Comment::getCreateTime));

        // 构建评论树
        Map<Integer, List<Comment>> childrenMap = list.stream()
                .filter(comment -> comment.getParentId() != null)
                .collect(Collectors.groupingBy(Comment::getParentId));

        List<Comment> result = list.stream()
                .filter(comment -> comment.getParentId() == null)
                .peek(comment -> setChildrenRecursively(comment, childrenMap))
                .collect(Collectors.toList());
        return new PageResult<>((long) list.size(), (long) pageNum, result);
    }

    @Async
    @Override
    @Transactional
    public void deleteById(String id) {
        // 递归删除指定评论及其所有子评论
        recursiveDelete(id);
    }

    private void recursiveDelete(String parentId) {
        // 查询所有以指定评论为父评论的子评论
        List<Comment> children = this.list(new LambdaQueryWrapper<Comment>()
                .eq(Comment::getParentId, parentId));

        // 递归删除每个子评论及其子评论
        for (Comment child : children) {
            recursiveDelete(String.valueOf(child.getId()));
        }

        // 删除当前评论
        this.removeById(parentId);
    }

    private void setChildrenRecursively(Comment parentComment, Map<Integer, List<Comment>> childrenMap) {
        List<Comment> children = childrenMap.getOrDefault(parentComment.getId(), List.of());
        parentComment.setChildren(children);
        children.forEach(child -> setChildrenRecursively(child, childrenMap));
    }
}
