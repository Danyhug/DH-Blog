package top.zzf4.blog.utils.ip;

import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * {
 *     "code": 200,
 *     "msg": "success",
 *     "data": {
 *         "address": "中国 北京 北京 联通",
 *         "ip": "123.123.123.123"
 *     }
 * }
 */
public class Csdn {

    @JsonProperty("code")
    private int code;

    @JsonProperty("msg")
    private String msg;

    @JsonProperty("data")
    private Data data;

    // 内部静态类 Data
    public static class Data {
        @JsonProperty("address")
        private String address;

        @JsonProperty("ip")
        private String ip;

        // Getters and Setters
        public String getAddress() {
            // 格式"中国 北京 北京 联通"
            // 转为 北京-北京|联通
            // 如果包含中国
            if (address.contains("中国")) {
                String[] split = address.split(" ");
                return split[3] + "/" + split[1] + split[2];
            }
            return this.address;
        }

        public void setAddress(String address) {
            this.address = address;
        }

        public String getIp() {
            return ip;
        }

        public void setIp(String ip) {
            this.ip = ip;
        }

        @Override
        public String toString() {
            return "Data{" +
                    "address='" + address + '\'' +
                    ", ip='" + ip + '\'' +
                    '}';
        }
    }

    // Getters and Setters
    public int getCode() {
        return code;
    }

    public void setCode(int code) {
        this.code = code;
    }

    public String getMsg() {
        return msg;
    }

    public void setMsg(String msg) {
        this.msg = msg;
    }

    public Data getData() {
        return data;
    }

    public void setData(Data data) {
        this.data = data;
    }

    @Override
    public String toString() {
        return "Csdn{" +
                "code=" + code +
                ", msg='" + msg + '\'' +
                ", data=" + data +
                '}';
    }
}
