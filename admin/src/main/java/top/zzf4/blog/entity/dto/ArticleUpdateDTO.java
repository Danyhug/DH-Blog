package top.zzf4.blog.entity.dto;

import lombok.Data;

import java.util.List;

@Data
public class ArticleUpdateDTO {

    private Long id;

    private String title;

    private String content;

    private Integer categoryId;

    private byte wordNum; // TINYINT映射为byte

    private List<String> tags;
}
