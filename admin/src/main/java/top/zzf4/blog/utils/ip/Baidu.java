package top.zzf4.blog.utils.ip;

import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.List;

/**
 * {
 *     "status": "0",
 *     "t": "",
 *     "set_cache_time": "",
 *     "data": [
 *         {
 *             "ExtendedLocation": "",
 *             "OriginQuery": "123.123.123.123",
 *             "SchemaVer": "",
 *             "appinfo": "",
 *             "disp_type": 0,
 *             "fetchkey": "123.123.123.123",
 *             "location": "北京市北京市 联通",
 *             "origip": "123.123.123.123",
 *             "origipquery": "123.123.123.123",
 *             "resourceid": "6006",
 *             "role_id": 0,
 *             "schemaID": "",
 *             "shareImage": 1,
 *             "showLikeShare": 1,
 *             "showlamp": "1",
 *             "strategyData": {},
 *             "titlecont": "IP地址查询",
 *             "tplt": "ip"
 *         }
 *     ]
 * }
 */
public class Baidu {

    @JsonProperty("status")
    private String status;

    @JsonProperty("t")
    private String t;

    @JsonProperty("set_cache_time")
    private String setCacheTime;

    @JsonProperty("data")
    private List<Data> data;

    // Getters and Setters
    public String getStatus() {
        return status;
    }

    public void setStatus(String status) {
        this.status = status;
    }

    public String getT() {
        return t;
    }

    public void setT(String t) {
        this.t = t;
    }

    public String getSetCacheTime() {
        return setCacheTime;
    }

    public void setSetCacheTime(String setCacheTime) {
        this.setCacheTime = setCacheTime;
    }

    public List<Data> getData() {
        return data;
    }

    public void setData(List<Data> data) {
        this.data = data;
    }

    @Override
    public String toString() {
        return "Baidu{" +
                "status='" + status + '\'' +
                ", t='" + t + '\'' +
                ", setCacheTime='" + setCacheTime + '\'' +
                ", data=" + data +
                '}';
    }

    public static class Data {

        @JsonProperty("ExtendedLocation")
        private String extendedLocation;

        @JsonProperty("OriginQuery")
        private String originQuery;

        @JsonProperty("SchemaVer")
        private String schemaVer;

        @JsonProperty("appinfo")
        private String appinfo;

        @JsonProperty("disp_type")
        private int dispType;

        @JsonProperty("fetchkey")
        private String fetchKey;

        @JsonProperty("location")
        private String location;

        @JsonProperty("origip")
        private String origip;

        @JsonProperty("origipquery")
        private String origipquery;

        @JsonProperty("resourceid")
        private String resourceid;

        @JsonProperty("role_id")
        private int roleId;

        @JsonProperty("schemaID")
        private String schemaID;

        @JsonProperty("shareImage")
        private int shareImage;

        @JsonProperty("showLikeShare")
        private int showLikeShare;

        @JsonProperty("showlamp")
        private String showlamp;

        @JsonProperty("strategyData")
        private Object strategyData;

        @JsonProperty("titlecont")
        private String titlecont;

        @JsonProperty("tplt")
        private String tplt;

        // Getters and Setters
        public String getExtendedLocation() {
            return extendedLocation;
        }

        public void setExtendedLocation(String extendedLocation) {
            this.extendedLocation = extendedLocation;
        }

        public String getOriginQuery() {
            return originQuery;
        }

        public void setOriginQuery(String originQuery) {
            this.originQuery = originQuery;
        }

        public String getSchemaVer() {
            return schemaVer;
        }

        public void setSchemaVer(String schemaVer) {
            this.schemaVer = schemaVer;
        }

        public String getAppinfo() {
            return appinfo;
        }

        public void setAppinfo(String appinfo) {
            this.appinfo = appinfo;
        }

        public int getDispType() {
            return dispType;
        }

        public void setDispType(int dispType) {
            this.dispType = dispType;
        }

        public String getFetchKey() {
            return fetchKey;
        }

        public void setFetchKey(String fetchKey) {
            this.fetchKey = fetchKey;
        }

        public String getLocation() {
            return location;
        }

        public void setLocation(String location) {
            this.location = location;
        }

        public String getOrigip() {
            return origip;
        }

        public void setOrigip(String origip) {
            this.origip = origip;
        }

        public String getOrigipquery() {
            return origipquery;
        }

        public void setOrigipquery(String origipquery) {
            this.origipquery = origipquery;
        }

        public String getResourceid() {
            return resourceid;
        }

        public void setResourceid(String resourceid) {
            this.resourceid = resourceid;
        }

        public int getRoleId() {
            return roleId;
        }

        public void setRoleId(int roleId) {
            this.roleId = roleId;
        }

        public String getSchemaID() {
            return schemaID;
        }

        public void setSchemaID(String schemaID) {
            this.schemaID = schemaID;
        }

        public int getShareImage() {
            return shareImage;
        }

        public void setShareImage(int shareImage) {
            this.shareImage = shareImage;
        }

        public int getShowLikeShare() {
            return showLikeShare;
        }

        public void setShowLikeShare(int showLikeShare) {
            this.showLikeShare = showLikeShare;
        }

        public String getShowlamp() {
            return showlamp;
        }

        public void setShowlamp(String showlamp) {
            this.showlamp = showlamp;
        }

        public Object getStrategyData() {
            return strategyData;
        }

        public void setStrategyData(Object strategyData) {
            this.strategyData = strategyData;
        }

        public String getTitlecont() {
            return titlecont;
        }

        public void setTitlecont(String titlecont) {
            this.titlecont = titlecont;
        }

        public String getTplt() {
            return tplt;
        }

        public void setTplt(String tplt) {
            this.tplt = tplt;
        }

        @Override
        public String toString() {
            return "Data{" +
                    "extendedLocation='" + extendedLocation + '\'' +
                    ", originQuery='" + originQuery + '\'' +
                    ", schemaVer='" + schemaVer + '\'' +
                    ", appinfo='" + appinfo + '\'' +
                    ", dispType=" + dispType +
                    ", fetchKey='" + fetchKey + '\'' +
                    ", location='" + location + '\'' +
                    ", origip='" + origip + '\'' +
                    ", origipquery='" + origipquery + '\'' +
                    ", resourceid='" + resourceid + '\'' +
                    ", roleId=" + roleId +
                    ", schemaID='" + schemaID + '\'' +
                    ", shareImage=" + shareImage +
                    ", showLikeShare=" + showLikeShare +
                    ", showlamp='" + showlamp + '\'' +
                    ", strategyData=" + strategyData +
                    ", titlecont='" + titlecont + '\'' +
                    ", tplt='" + tplt + '\'' +
                    '}';
        }
    }
}
