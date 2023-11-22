// @ts-ignore
import Cookies from "js-cookie";

export const setCookie = (key: string, value: any, option = null) => {
  Cookies.set(key, JSON.stringify(value), option);
};

export const getCookie = (key: string) => {
  return JSON.parse(Cookies.get(key) || null);
};

export const removeCookie = async (key: string) => {
  Cookies.remove(key);
};
