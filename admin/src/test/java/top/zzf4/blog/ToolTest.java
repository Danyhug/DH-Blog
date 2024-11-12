package top.zzf4.blog;

import org.junit.jupiter.api.Test;
import top.zzf4.blog.utils.JwtUtils;
import top.zzf4.blog.utils.Tools;

public class ToolTest {
    @Test
    public void test() {
        String password = JwtUtils.encodeByBCrypt("1234");
        System.out.println(password);
        System.out.println(JwtUtils.verifyByBCrypt("1234", password));
    }

    @Test
    public void testIpCity() {
        System.out.println(Tools.getIpCity("127.0.0.1"));
    }
}
