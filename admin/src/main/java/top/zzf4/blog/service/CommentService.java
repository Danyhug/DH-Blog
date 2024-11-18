package top.zzf4.blog.service;

import com.baomidou.mybatisplus.extension.service.IService;
import top.zzf4.blog.entity.model.Comment;

public interface CommentService extends IService<Comment> {
    // 添加评论
    boolean addComment(Comment comment);
}
