package top.zzf4.blog.utils.ip;
import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * {
 *     "ip": "123.123.123.123",
 *     "pro": "北京市",
 *     "proCode": "110000",
 *     "city": "北京市",
 *     "cityCode": "110000",
 *     "region": "顺义区",
 *     "regionCode": "110113",
 *     "addr": "北京市顺义区 联通",
 *     "regionNames": "",
 *     "err": ""
 * }
 */
public class PConline {

    @JsonProperty("ip")
    private String ip;

    @JsonProperty("pro")
    private String pro;

    @JsonProperty("proCode")
    private String proCode;

    @JsonProperty("city")
    private String city;

    @JsonProperty("cityCode")
    private String cityCode;

    @JsonProperty("region")
    private String region;

    @JsonProperty("regionCode")
    private String regionCode;

    @JsonProperty("addr")
    private String addr;

    @JsonProperty("regionNames")
    private String regionNames;

    @JsonProperty("err")
    private String err;

    // Getters and Setters
    public String getIp() {
        return ip;
    }

    public void setIp(String ip) {
        this.ip = ip;
    }

    public String getPro() {
        return pro;
    }

    public void setPro(String pro) {
        this.pro = pro;
    }

    public String getProCode() {
        return proCode;
    }

    public void setProCode(String proCode) {
        this.proCode = proCode;
    }

    public String getCity() {
        return city;
    }

    public void setCity(String city) {
        this.city = city;
    }

    public String getCityCode() {
        return cityCode;
    }

    public void setCityCode(String cityCode) {
        this.cityCode = cityCode;
    }

    public String getRegion() {
        return region;
    }

    public void setRegion(String region) {
        this.region = region;
    }

    public String getRegionCode() {
        return regionCode;
    }

    public void setRegionCode(String regionCode) {
        this.regionCode = regionCode;
    }

    public String getAddr() {
        return addr;
    }

    public void setAddr(String addr) {
        this.addr = addr;
    }

    public String getRegionNames() {
        return regionNames;
    }

    public void setRegionNames(String regionNames) {
        this.regionNames = regionNames;
    }

    public String getErr() {
        return err;
    }

    public void setErr(String err) {
        this.err = err;
    }

    @Override
    public String toString() {
        return "PConline{" +
                "ip='" + ip + '\'' +
                ", pro='" + pro + '\'' +
                ", proCode='" + proCode + '\'' +
                ", city='" + city + '\'' +
                ", cityCode='" + cityCode + '\'' +
                ", region='" + region + '\'' +
                ", regionCode='" + regionCode + '\'' +
                ", addr='" + addr + '\'' +
                ", regionNames='" + regionNames + '\'' +
                ", err='" + err + '\'' +
                '}';
    }
}
