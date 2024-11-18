package top.zzf4.blog.controller;

import lombok.extern.log4j.Log4j2;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;
import top.zzf4.blog.entity.AjaxResult;
import top.zzf4.blog.entity.model.Comment;
import top.zzf4.blog.entity.vo.PageResult;
import top.zzf4.blog.service.CommentService;
import top.zzf4.blog.utils.Tools;

import jakarta.servlet.http.HttpServletRequest;

@Log4j2
@RestController
@RequestMapping("/comment")
@io.swagger.v3.oas.annotations.tags.Tag(name = "评论控制器")
public class CommentController {

    @Autowired
    private CommentService commentService;

    @PostMapping
    public AjaxResult<String> addComment(@RequestBody Comment comment, HttpServletRequest request) {
        String ua = Tools.parseUserAgent(request.getHeader("User-Agent"));
        comment.setUa(ua);
        commentService.addComment(comment);
        System.out.println(comment);
        return AjaxResult.success("评论成功！");
    }

    @GetMapping("/{articleId}")
    public AjaxResult<PageResult<Comment>> getCommentList(@PathVariable Long articleId) {
        PageResult<Comment> commentList = commentService.getCommentList(articleId, 100, 1);
        return AjaxResult.success(commentList);
    }
}
