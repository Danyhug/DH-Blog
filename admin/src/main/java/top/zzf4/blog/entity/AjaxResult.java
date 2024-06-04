package top.zzf4.blog.entity;

import lombok.Data;

import java.io.Serializable;

@Data
public class AjaxResult<T> implements Serializable {
    private Integer code; //编码：1成功，0和其它数字为失败
    private String msg; //错误信息
    private T data; //数据

    public static <T> AjaxResult<T> success() {
        AjaxResult<T> result = new AjaxResult<>();
        result.setCode(1);
        return result;
    }

    public static <T> AjaxResult<T> success(T data) {
        AjaxResult<T> result = new AjaxResult<>();
        result.setCode(1);
        result.setData(data);
        return result;
    }

    public static <T> AjaxResult<T> error(String msg) {
        AjaxResult<T> result = new AjaxResult<>();
        result.setCode(0);
        result.setMsg(msg);
        return result;
    }
}
