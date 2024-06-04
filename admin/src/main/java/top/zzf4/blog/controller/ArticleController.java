package top.zzf4.blog.controller;

import lombok.extern.log4j.Log4j2;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import top.zzf4.blog.entity.AjaxResult;
import top.zzf4.blog.entity.dto.ArticleDetailDTO;

@RestController
@Log4j2
@RequestMapping("/article")
public class ArticleController {
    @GetMapping("/detail")
    public AjaxResult<ArticleDetailDTO> detail(@RequestParam String id) {
        log.info("获取文章详情 {}", id);
        return AjaxResult.success();
    }
}
