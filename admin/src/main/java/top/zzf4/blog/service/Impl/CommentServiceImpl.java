package top.zzf4.blog.service.Impl;

import com.baomidou.mybatisplus.extension.service.impl.ServiceImpl;
import org.springframework.stereotype.Service;
import top.zzf4.blog.entity.model.Comment;
import top.zzf4.blog.mapper.CommentMapper;
import top.zzf4.blog.service.CommentService;

@Service
public class CommentServiceImpl extends ServiceImpl<CommentMapper, Comment> implements CommentService {
    @Override
    public boolean addComment(Comment comment) {
        return this.save(comment);
    }
}
