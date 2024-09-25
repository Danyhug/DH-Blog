package top.zzf4.blog.controller;

import io.swagger.v3.oas.annotations.tags.Tag;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@Tag(name = "后台管理控制器")
@RequestMapping("/admin")
public class AdminController {
}
