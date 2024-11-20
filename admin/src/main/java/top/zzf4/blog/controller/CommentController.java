package top.zzf4.blog.controller;

import jakarta.servlet.http.HttpServletRequest;
import lombok.extern.log4j.Log4j2;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;
import top.zzf4.blog.entity.AjaxResult;
import top.zzf4.blog.entity.model.Comment;
import top.zzf4.blog.entity.vo.PageResult;
import top.zzf4.blog.service.CommentService;
import top.zzf4.blog.utils.Tools;

@Log4j2
@RestController
@RequestMapping("/comment")
@io.swagger.v3.oas.annotations.tags.Tag(name = "评论控制器")
public class CommentController {

    @Autowired
    private CommentService commentService;

    /**
     * 添加评论
     * @param comment
     * @param request
     * @return
     */
    @PostMapping
    public AjaxResult<String> addComment(@RequestBody Comment comment, HttpServletRequest request) {
        String ua = Tools.parseUserAgent(request.getHeader("User-Agent"));
        comment.setUa(ua);
        comment.setIsAdmin(false);
        commentService.addComment(comment);
        System.out.println(comment);
        return AjaxResult.success("评论成功！");
    }

    /**
     * 查询评论列表
     * @param articleId
     * @return
     */
    @GetMapping("/{articleId}")
    public AjaxResult<PageResult<Comment>> getCommentList(@PathVariable Long articleId) {
        PageResult<Comment> commentList = commentService.getCommentListByArticle(articleId, 100, 1);
        return AjaxResult.success(commentList);
    }

}
