package top.zzf4.blog;

import org.junit.jupiter.api.Test;
import top.zzf4.blog.utils.Tools;

public class ToolTest {
    @Test
    public void test() {
        String password = Tools.encodeByBCrypt("1234");
        System.out.println(password);
        System.out.println(Tools.verifyByBCrypt("1234", password));
    }
}
