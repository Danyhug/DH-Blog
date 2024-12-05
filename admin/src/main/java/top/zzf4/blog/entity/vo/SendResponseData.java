package top.zzf4.blog.entity.vo;

import jakarta.servlet.http.HttpServletResponse;
import lombok.AllArgsConstructor;
import lombok.Data;

@Data
@AllArgsConstructor
public class SendResponseData {
    int code;
    String msg;
    HttpServletResponse response;
}
