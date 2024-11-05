package top.zzf4.blog.aop;

import java.lang.annotation.ElementType;
import java.lang.annotation.Retention;
import java.lang.annotation.RetentionPolicy;
import java.lang.annotation.Target;

@Target(ElementType.METHOD)
@Retention(RetentionPolicy.RUNTIME)
public @interface Limit{
    // 缓存的存在时间
    int time() default 60;
    // 允许的最大次数
    int num() default 30;
    // 返回文本
    String msg() default "请求过于频繁！";
}
