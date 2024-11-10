package top.zzf4.blog.utils.ip;

import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * {
 *     "status": "success",
 *     "country": "中国",
 *     "countryCode": "CN",
 *     "region": "BJ",
 *     "regionName": "北京市",
 *     "city": "北京",
 *     "zip": "",
 *     "lat": 39.911,
 *     "lon": 116.395,
 *     "timezone": "Asia/Shanghai",
 *     "isp": "China Unicom Beijing Province Network",
 *     "org": "",
 *     "as": "AS4808 China Unicom Beijing Province Network",
 *     "query": "123.123.123.123"
 */
public class IPapi {

    @JsonProperty("status")
    private String status;

    @JsonProperty("country")
    private String country;

    @JsonProperty("countryCode")
    private String countryCode;

    @JsonProperty("region")
    private String region;

    @JsonProperty("regionName")
    private String regionName;

    @JsonProperty("city")
    private String city;

    @JsonProperty("zip")
    private String zip;

    @JsonProperty("lat")
    private double lat;

    @JsonProperty("lon")
    private double lon;

    @JsonProperty("timezone")
    private String timezone;

    @JsonProperty("isp")
    private String isp;

    @JsonProperty("org")
    private String org;

    @JsonProperty("as")
    private String as;

    @JsonProperty("query")
    private String query;

    // Getters and Setters
    public String getStatus() {
        return status;
    }

    public void setStatus(String status) {
        this.status = status;
    }

    public String getCountry() {
        return country;
    }

    public void setCountry(String country) {
        this.country = country;
    }

    public String getCountryCode() {
        return countryCode;
    }

    public void setCountryCode(String countryCode) {
        this.countryCode = countryCode;
    }

    public String getRegion() {
        return region;
    }

    public void setRegion(String region) {
        this.region = region;
    }

    public String getRegionName() {
        return regionName;
    }

    public void setRegionName(String regionName) {
        this.regionName = regionName;
    }

    public String getCity() {
        return city;
    }

    public void setCity(String city) {
        this.city = city;
    }

    public String getZip() {
        return zip;
    }

    public void setZip(String zip) {
        this.zip = zip;
    }

    public double getLat() {
        return lat;
    }

    public void setLat(double lat) {
        this.lat = lat;
    }

    public double getLon() {
        return lon;
    }

    public void setLon(double lon) {
        this.lon = lon;
    }

    public String getTimezone() {
        return timezone;
    }

    public void setTimezone(String timezone) {
        this.timezone = timezone;
    }

    public String getIsp() {
        return isp;
    }

    public void setIsp(String isp) {
        this.isp = isp;
    }

    public String getOrg() {
        return org;
    }

    public void setOrg(String org) {
        this.org = org;
    }

    public String getAs() {
        return as;
    }

    public void setAs(String as) {
        this.as = as;
    }

    public String getQuery() {
        return query;
    }

    public void setQuery(String query) {
        this.query = query;
    }

    @Override
    public String toString() {
        return "IPapi{" +
                "status='" + status + '\'' +
                ", country='" + country + '\'' +
                ", countryCode='" + countryCode + '\'' +
                ", region='" + region + '\'' +
                ", regionName='" + regionName + '\'' +
                ", city='" + city + '\'' +
                ", zip='" + zip + '\'' +
                ", lat=" + lat +
                ", lon=" + lon +
                ", timezone='" + timezone + '\'' +
                ", isp='" + isp + '\'' +
                ", org='" + org + '\'' +
                ", as='" + as + '\'' +
                ", query='" + query + '\'' +
                '}';
    }
}
