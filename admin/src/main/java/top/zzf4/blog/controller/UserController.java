package top.zzf4.blog.controller;

import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import jakarta.servlet.http.HttpServletRequest;
import lombok.extern.log4j.Log4j2;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;
import top.zzf4.blog.aop.Limit;
import top.zzf4.blog.entity.AjaxResult;
import top.zzf4.blog.entity.model.User;
import top.zzf4.blog.service.UserService;
import top.zzf4.blog.utils.Tools;

@Tag(name = "用户控制器")
@Log4j2
@RestController
@RequestMapping("/user")
public class UserController {
    @Autowired
    private UserService userService;

    @Limit
    @Operation(summary = "用户登录")
    @PostMapping("/login")
    public AjaxResult<String> login(@RequestBody User user) {
        return AjaxResult.success(userService.login(user.getUsername(), user.getPassword()));
    }

    @Limit
    @Operation(summary = "用户校验")
    @PostMapping("/check")
    public AjaxResult<String> check() {
        return AjaxResult.success("Success");
    }

    @Limit
    @Operation(summary = "用户在线状态监测")
    @GetMapping("/heart")
    public AjaxResult<String> heart(HttpServletRequest request) {
        userService.heart(Tools.getClientIp(request));
        return AjaxResult.success("咚咚咚 ~ 咚咚咚 ~" + userService.getOnlineNum());
    }
}
