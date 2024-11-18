package top.zzf4.blog.entity.model;

import com.baomidou.mybatisplus.annotation.*;
import com.fasterxml.jackson.annotation.JsonFormat;
import lombok.Data;

import java.time.LocalDateTime;
import java.util.List;

@Data
@TableName("comments")
public class Comment {

    @TableId(value = "id", type = IdType.AUTO)
    private Integer id;

    @TableField("article_id")
    private Integer articleId;

    @TableField("author")
    private String author;

    @TableField("email")
    private String email;

    @TableField("content")
    private String content;

    @TableField("public")
    private Boolean isPublic;

    @TableField(value = "create_time", fill = FieldFill.INSERT)
    private LocalDateTime createTime;

    @TableField("parent_id")
    private Integer parentId;

    @TableField("ua")
    private String ua;

    @TableField("admin")
    private Boolean isAdmin;

    @TableField(exist = false)
    private List<Comment> children;
}
