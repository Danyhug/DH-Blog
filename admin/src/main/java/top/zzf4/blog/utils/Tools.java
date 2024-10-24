package top.zzf4.blog.utils;

import cn.hutool.crypto.digest.BCrypt;

public class Tools {
    public static String encodeByBCrypt(String str) {
        return BCrypt.hashpw(str, BCrypt.gensalt());
    }

    public static boolean verifyByBCrypt(String str, String hash) {
        return BCrypt.checkpw(str, hash);
    }
}
