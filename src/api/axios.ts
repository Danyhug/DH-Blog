import axios, { AxiosResponse, InternalAxiosRequestConfig } from "axios"
import { SERVER_URL } from '@/types/Constant.ts'
import router from "@/router"

// 返回值类型
interface AjaxResult<T> {
  code: number
  msg: string
  data: T
}

const request = axios.create({
  baseURL: SERVER_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json;charset=UTF-8',
  },
})

// 添加请求拦截器
request.interceptors.request.use(
  function (config: InternalAxiosRequestConfig) {
    // 在发送请求之前做些什么
    config.headers.Authorization = localStorage.getItem('token') || '';
    return config;
  },
  function (error) {
    // 对请求错误做些什么
    return Promise.reject(error);
  }
);

// 添加响应拦截器
request.interceptors.response.use(
  function (response: AxiosResponse<AjaxResult<any>>) {
    // 2xx 范围内的状态码都会触发该函数。
    // 这里假设后端返回的数据结构中有一个code字段用于标识操作结果
    if (response.data.code === 1) {
      // 当code为1时，直接返回data部分
      return response.data.data;
    } else {
      // 如果code不是1，你可以根据需要处理错误，比如抛出异常或返回特定错误信息
      ElMessage.error(response.data.msg);
      return Promise.reject(new Error(response.data.msg || 'Error'));
    }
  },
  function (error) {
    // 超出 2xx 范围的状态码都会触发该函数。
    // 这里可以处理网络错误、超时等
    if (error.status === 401) {
      localStorage.removeItem('token')
      router.replace({ name: 'Login' })
    }

    return Promise.reject(error);
  }
);

export default request;