package top.zzf4.blog.service;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.extension.service.IService;
import top.zzf4.blog.entity.model.Comment;
import top.zzf4.blog.entity.vo.PageResult;

public interface CommentService extends IService<Comment> {
    // 添加评论
    boolean addComment(Comment comment);

    // 查看指定文章的评论
    PageResult<Comment> getCommentListByArticle(Long articleId, int pageSize, int pageNum);

    // 查看所有文章的评论
    PageResult<Comment> getCommentList(int pageSize, int pageNum, LambdaQueryWrapper<Comment> queryWrapper);

    void deleteById(String id);
}
