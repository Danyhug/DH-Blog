package top.zzf4.blog.controller;

import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import lombok.extern.log4j.Log4j2;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;
import top.zzf4.blog.entity.AjaxResult;
import top.zzf4.blog.entity.model.User;
import top.zzf4.blog.service.UserService;

@Tag(name = "用户控制器")
@Log4j2
@RestController
@RequestMapping("/user")
public class UserController {
    @Autowired
    private UserService userService;

    @Operation(summary = "用户登录")
    @PostMapping("/login")
    public AjaxResult<String> login(@RequestBody User user) {
        return AjaxResult.success(userService.login(user.getUsername(), user.getPassword()));
    }

    @Operation(summary = "用户校验")
    @PostMapping("/check")
    public AjaxResult<String> check() {
        return AjaxResult.success("Success");
    };

}
