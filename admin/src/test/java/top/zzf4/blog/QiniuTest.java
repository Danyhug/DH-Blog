package top.zzf4.blog;

import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import top.zzf4.blog.service.Impl.QiniuServiceImpl;

import java.util.Arrays;

@SpringBootTest
public class QiniuTest {

    @Autowired
    private QiniuServiceImpl qiniuService;

    @Test
    void getFileList() {
        System.out.println(Arrays.toString(qiniuService.getFileList("defaultArticleImg")));
    }

    @Test
    void init() {
        qiniuService.initCache();
    }

    @Test
    void getRandomImage() {
        for (int i = 0; i < 100; i++) {
            System.out.println(qiniuService.getRandomDefaultImage());
        }
    }
}
