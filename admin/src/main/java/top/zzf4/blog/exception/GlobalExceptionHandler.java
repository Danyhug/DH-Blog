package top.zzf4.blog.exception;

import lombok.extern.log4j.Log4j2;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;
import top.zzf4.blog.entity.AjaxResult;

import java.sql.SQLIntegrityConstraintViolationException;

// 全局异常处理
@Log4j2
@RestControllerAdvice
public class GlobalExceptionHandler {

    /**
     * 捕获业务异常
     */
    // @ExceptionHandler
    // public AjaxResult<Void> exceptionHandler(Exception ex){
    //     log.error("异常信息：{}", ex.getMessage());
    //     log.error(ex);
    //     return AjaxResult.error(ex.getMessage());
    // }

    @ExceptionHandler
    public AjaxResult<String> exceptionHandler(SQLIntegrityConstraintViolationException ex) {
        return AjaxResult.error(ex.getMessage());
    }

    @ExceptionHandler
    public AjaxResult<String> exceptionHandler(RuntimeException ex) {
        log.error("异常信息：{}", ex.getMessage());
        return AjaxResult.error(ex.getMessage());
    }
}
