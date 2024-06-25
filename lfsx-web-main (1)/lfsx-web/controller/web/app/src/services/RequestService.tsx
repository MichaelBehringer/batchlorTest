import axios, {AxiosError, AxiosProgressEvent, AxiosRequestConfig, AxiosResponse} from 'axios';
import SecurityHelper from './SecuriyHelper';

const baseURL = '/api'

interface Options {
	method?: "get" | "post" | "patch" | "delete",
	acceptCookieSetHeader?: boolean,

	/* Timeout in mili seconds */
	timeout?: number,

	/* Returns the progress of the upload */
	onUploadProgress?: ((progressEvent: AxiosProgressEvent) => void)
}

interface StandardResponse {
	data: any,
	status: {
		code: number
	},
	headers: any,
	errorMessage?: any
}

const getPath = (path: string): string => {
	configureAxios()
	return path
}
const getParams = (params: any): any => {
	if (params === null || params === undefined)  params = { "userIdentifier": SecurityHelper.getUserIdentification() }
	else params["userIdentifier"] = SecurityHelper.getUserIdentification()

	return params
}

/**
 * Makes an request to the REST-API
 * 
 * @param path      the path to the resource
 * @param params    the request params ("url params")
 * @param body      the request body
 * @param header    the request header
 * @param options   options for the request
 * 
 * @returns 
 */
const getRequest = (path: string, params: any, body: any, headers: {[key in string]: any}, options: Options): Promise<StandardResponse> => {
	configureAxios();

	if (options === null) options = {}

	// set default options
	options.method ??= "get"

	return new Promise((resolve) => {
		axios.request({
			method: options.method,
			url: path,
			params: getParams(params),
			data: body,
			headers: headers,
			withCredentials: options.acceptCookieSetHeader,
			timeout: options.timeout,
			onUploadProgress: options.onUploadProgress
		}).then((response: AxiosResponse) => {
			const standardResponse = getStandardResponse(response)
			resolve(standardResponse)
		}).catch((err: AxiosError) => {
			resolve(getErrorResponse(err))	// no reject
		})
	})
}

const configureAxios = () => {
	axios.defaults.baseURL = baseURL;
	//axios.defaults.headers.common['Authorization'] = AUTH_TOKEN;
	axios.defaults.headers.post['Content-Type'] = 'application/x-www-form-urlencoded';

	axios.interceptors.request.use((config) => {
		config.params = config.params || {}
		config.params['userIdentifier'] = SecurityHelper.getUserIdentification()
		return config
	})
}

const getStandardResponse = (response: AxiosResponse | any, cached?: boolean): StandardResponse => {
	return {
		data: response.data,
		status: {
			code: cached ? 200 : response.request.status
		},
		headers: cached ? {cache: "cache"} : response.headers
	}
}

const getErrorResponse = (response: AxiosError): StandardResponse => {
	return {
		data: response.response?.data,
		status: {
			code: response.response == null ? 500 : response.response?.status
		},
		headers: response.response?.headers,
		errorMessage: response.message
	}

}

export default getRequest //{ getRequest }
export type {Options, StandardResponse};
export {RequestHelper, configureAxios};

const axiosInstance = axios.create();
axiosInstance.interceptors.response.use((response: AxiosResponse<unknown, unknown>): AxiosResponse<unknown, unknown> => {
	return response;
}, (error: AxiosError): Promise<never> => {
	console.error(error?.response?.status ?? "" + (error?.response?.data));
	return Promise.reject(error);
});
axiosInstance.defaults.baseURL = baseURL;

class RequestHelper {

	private static cache = new Map<string, unknown[]>();
	private static cacheIt = new Set<string>([]);
	private static clearIt = new Map<string, string>([]);

	public static get = (path: string, params?: unknown): Promise<StandardResponse> => {
		path = getPath(path)
		if (RequestHelper.cache.has(path)) {
			return new Promise<StandardResponse>((resolve) => {
				resolve(getStandardResponse({data: RequestHelper.cache.get(path)}, true))
			});
		} else {
			return new Promise<StandardResponse>((resolve) => {
				axios.get<unknown>(path, { params: getParams(params) })
					.then((response: AxiosResponse) => {
						if (RequestHelper.cacheIt.has(path)) {
							RequestHelper.cache.set(path, response.data);
						}
						resolve(getStandardResponse(response));
					})
					.catch((error: AxiosError) => {
						RequestHelper.checkAuthorization(error, path);
						resolve(getErrorResponse(error))	// no reject
					});
			});
		}
	};

	public static post = (path: string, data: unknown, options?: AxiosRequestConfig<unknown>): Promise<StandardResponse> => {
		path = getPath(path)
		if (options === null || options === undefined) options = {  } 
		options.params = getParams(options?.params)

		return new Promise<StandardResponse>((resolve) => {
			axiosInstance.post<unknown>(path, data, options )
				.then((response: AxiosResponse) => {
					resolve(getStandardResponse(response));
				})
				.catch((error: AxiosError) => {
					RequestHelper.checkAuthorization(error);
					resolve(getErrorResponse(error))	// no reject
				});
		});
	};

	public static patch = (path: string, data: unknown, options?: AxiosRequestConfig<unknown>): Promise<StandardResponse> => {
		path = getPath(path)
		if (options === null || options === undefined) options = {  } 
		options.params = getParams(options?.params)

		return new Promise<StandardResponse>((resolve) => {
			axiosInstance.patch<unknown>(path, data, options)
				.then((response: AxiosResponse) => {
					resolve(getStandardResponse(response));
				})
				.catch((error: AxiosError) => {
					RequestHelper.checkAuthorization(error);
					resolve(getErrorResponse(error))	// no reject
				});
		});
	};

	public static put = (path: string, data: unknown, options?: AxiosRequestConfig<unknown>): Promise<StandardResponse> => {
		path = getPath(path)
		if (options === null || options === undefined) options = {  } 
		options.params = getParams(options?.params)

		return new Promise<StandardResponse>((resolve) => {
			axiosInstance.put<unknown>(path, data, options)
				.then((response: AxiosResponse) => {
					resolve(getStandardResponse(response));
				})
				.catch((error: AxiosError) => {
					RequestHelper.checkAuthorization(error);
					resolve(getErrorResponse(error))	// no reject
				});
		});
	};

	public static delete = (path: string, options?: AxiosRequestConfig<unknown>): Promise<StandardResponse> => {
		path = getPath(path)
		if (options === null || options === undefined) options = {  } 
		options.params = getParams(options?.params)

		return new Promise<StandardResponse>((resolve) => {
			axiosInstance.delete<unknown>(path, options)
				.then((response: AxiosResponse) => {
					resolve(getStandardResponse(response));
				})
				.catch((error: AxiosError) => {
					RequestHelper.checkAuthorization(error);
					resolve(getErrorResponse(error))	// no reject
				});
		});
	};

	private static clearCache(path: string) {
		if (this.clearIt.has(path)) {
			const cachedPath = this.clearIt.get(path);
			if (cachedPath) {
				this.cache.delete(cachedPath);
			}
		}
	}

	private static checkAuthorization = (error: AxiosError, path?: string) => {
		if (path && path.startsWith(baseURL + "/isAuthenticated")) return;
		if (error.response && (error.response.status === 401 || error.response.status === 403)) {
			SecurityHelper.clearAllLoginCredentials()
			SecurityHelper.redirectToLogin()
		}
	};
}
